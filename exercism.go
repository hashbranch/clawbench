package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	exercismRepoURL = "https://github.com/Aider-AI/polyglot-benchmark/archive/refs/heads/main.zip"
	exercismSubdir  = "polyglot-benchmark-main/python/exercises/practice"
)

// ExercismBenchmark fetches and caches the Aider polyglot Python exercises,
// then generates ClawBench tasks from them.
type ExercismBenchmark struct {
	cacheDir string
}

// NewExercismBenchmark creates a benchmark using the given cache directory.
func NewExercismBenchmark(cacheDir string) *ExercismBenchmark {
	return &ExercismBenchmark{cacheDir: cacheDir}
}

// EnsureDownloaded fetches the exercises if not already cached.
func (e *ExercismBenchmark) EnsureDownloaded() error {
	marker := filepath.Join(e.cacheDir, ".downloaded")
	if _, err := os.Stat(marker); err == nil {
		return nil // already cached
	}

	fmt.Println("Downloading Exercism Python exercises from Aider polyglot benchmark...")
	resp, err := http.Get(exercismRepoURL)
	if err != nil {
		return fmt.Errorf("failed to download exercises: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read download: %w", err)
	}

	// Extract just the python/exercises/practice/ directory
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return fmt.Errorf("failed to open zip: %w", err)
	}

	for _, f := range reader.File {
		if !strings.HasPrefix(f.Name, exercismSubdir) {
			continue
		}

		relPath := strings.TrimPrefix(f.Name, exercismSubdir+"/")
		if relPath == "" {
			continue
		}

		destPath := filepath.Join(e.cacheDir, "exercises", relPath)

		if f.FileInfo().IsDir() {
			os.MkdirAll(destPath, 0755)
			continue
		}

		os.MkdirAll(filepath.Dir(destPath), 0755)
		rc, err := f.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(rc)
		rc.Close()
		if err != nil {
			continue
		}
		os.WriteFile(destPath, data, 0644)
	}

	// Write marker
	os.WriteFile(marker, []byte(time.Now().Format(time.RFC3339)), 0644)
	fmt.Println("Exercises cached.")
	return nil
}

// ListExercises returns the names of all available Python exercises.
func (e *ExercismBenchmark) ListExercises() ([]string, error) {
	exerciseDir := filepath.Join(e.cacheDir, "exercises")
	entries, err := os.ReadDir(exerciseDir)
	if err != nil {
		return nil, fmt.Errorf("exercises not found at %s: %w", exerciseDir, err)
	}

	var names []string
	for _, entry := range entries {
		if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
			names = append(names, entry.Name())
		}
	}
	return names, nil
}

// GenerateTask creates a ClawBench task for a single Exercism exercise.
// The task prompt includes the instructions and the stub file content.
// The evaluator runs the unittest test suite.
func (e *ExercismBenchmark) GenerateTask(exerciseName string) (*Task, error) {
	exerciseDir := filepath.Join(e.cacheDir, "exercises", exerciseName)

	// Read instructions
	instructions, err := os.ReadFile(filepath.Join(exerciseDir, ".docs", "instructions.md"))
	if err != nil {
		return nil, fmt.Errorf("no instructions for %s: %w", exerciseName, err)
	}

	// Find the stub file (snake_case.py)
	snakeName := strings.ReplaceAll(exerciseName, "-", "_")
	stubPath := filepath.Join(exerciseDir, snakeName+".py")
	stub, err := os.ReadFile(stubPath)
	if err != nil {
		return nil, fmt.Errorf("no stub file for %s: %w", exerciseName, err)
	}

	testFile := snakeName + "_test.py"
	testPath := filepath.Join(exerciseDir, testFile)
	if _, err := os.Stat(testPath); err != nil {
		return nil, fmt.Errorf("no test file for %s: %w", exerciseName, err)
	}

	// Build the prompt: instructions + stub + what to do
	prompt := fmt.Sprintf(`Solve this Exercism coding exercise. Write your solution to the file %s.

## Instructions

%s

## Starting code (%s)

%s

## Test file

The test file is %s in the current directory. You can run it with:
  python -m pytest %s -v

Write ONLY the solution file %s. Do not modify the test file.`,
		snakeName+".py",
		string(instructions),
		snakeName+".py",
		string(stub),
		testFile,
		testFile,
		snakeName+".py",
	)

	task := &Task{
		ID:         "exercism/" + exerciseName,
		Name:       fmt.Sprintf("Exercism: %s", exerciseName),
		Category:   "coding",
		Tags:       []string{"exercism", "aider-polyglot", "python"},
		Prompt:     prompt,
		TimeBudget: 120 * time.Second,
		Evaluators: []EvalConfig{
			{Type: "exec_check", Path: fmt.Sprintf("python -m pytest %s -v", testFile), Weight: 1.0},
			{Type: "cost", Weight: 0.3},
			{Type: "latency", Weight: 0.3},
		},
	}

	return task, nil
}

// GenerateAllTasks creates tasks for all available exercises.
func (e *ExercismBenchmark) GenerateAllTasks() ([]Task, error) {
	names, err := e.ListExercises()
	if err != nil {
		return nil, err
	}

	var tasks []Task
	for _, name := range names {
		task, err := e.GenerateTask(name)
		if err != nil {
			fmt.Printf("  Skipping %s: %s\n", name, err)
			continue
		}
		tasks = append(tasks, *task)
	}
	return tasks, nil
}

// WorkspaceDir returns the path to an exercise's directory (used as workspace).
func (e *ExercismBenchmark) WorkspaceDir(exerciseName string) string {
	return filepath.Join(e.cacheDir, "exercises", exerciseName)
}
