package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RunTask executes a single benchmark task against a connected Gateway.
func RunTask(ctx context.Context, client Backend, task Task, workspacePath string) RunResult {
	runID := generateRunID()
	start := time.Now()

	// Create a timeout context for this task
	taskCtx, cancel := context.WithTimeout(ctx, task.TimeBudget)
	defer cancel()

	// Seed setup files into workspace before running
	if len(task.SetupFiles) > 0 && workspacePath != "" {
		for _, sf := range task.SetupFiles {
			fullPath := filepath.Join(workspacePath, sf.Path)
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			if err := os.WriteFile(fullPath, []byte(sf.Content), 0644); err != nil {
				return RunResult{
					RunID:    runID,
					TaskID:   task.ID,
					Timestamp: time.Now(),
					IsError:  true,
					ErrorMsg: fmt.Sprintf("failed to seed file %s: %v", sf.Path, err),
				}
			}
		}
	}

	// Send prompt and capture response
	resp, err := client.SendPrompt(taskCtx, task.Prompt)
	wallClock := time.Since(start).Seconds()

	if err != nil {
		return RunResult{
			RunID:            runID,
			TaskID:           task.ID,
			Timestamp:        time.Now(),
			WallClockSeconds: wallClock,
			IsError:          true,
			ErrorMsg:         err.Error(),
			Config: ConfigMeta{
				Model:       resp.Model,
				Temperature: resp.Temperature,
			},
		}
	}

	// Run evaluators
	evalResults := Evaluate(task, resp, wallClock, workspacePath)

	// Build result
	result := RunResult{
		RunID:            runID,
		TaskID:           task.ID,
		Timestamp:        time.Now(),
		Correctness:      ComputeCorrectness(evalResults),
		ToolAccuracy:     ComputeToolAccuracy(evalResults),
		WallClockSeconds: wallClock,
		TotalTokens:      resp.Tokens.Total,
		NumToolCalls:     len(resp.ToolCalls),
		RawResponse:      resp.Text,
		EvalResults:      evalResults,
		Config: ConfigMeta{
			Model:       resp.Model,
			Temperature: resp.Temperature,
		},
	}

	// Compute cost
	for _, er := range evalResults {
		if er.Type == "cost" {
			result.CostUSD = er.Score
			break
		}
	}

	// Collect tools used
	for _, tc := range resp.ToolCalls {
		result.ToolsUsed = append(result.ToolsUsed, tc.Name)
	}

	return result
}

// RunAll executes all tasks, optionally repeating each N times.
// Returns all individual run results.
// RunAll executes all tasks. taskWorkspaces maps taskID -> workspace path for
// benchmarks that need per-task workspaces (e.g., exercism). Falls back to
// defaultWorkspace if no per-task workspace is set.
func RunAll(ctx context.Context, client Backend, tasks []Task, repeatCount int, defaultWorkspace string, taskWorkspaces map[string]string) []RunResult {
	if repeatCount < 1 {
		repeatCount = 1
	}

	var allResults []RunResult

	for i, task := range tasks {
		ws := defaultWorkspace
		if taskWorkspaces != nil {
			if tw, ok := taskWorkspaces[task.ID]; ok {
				ws = tw
			}
		}

		fmt.Printf("  [%d/%d] %s", i+1, len(tasks), task.Name)
		for j := 0; j < repeatCount; j++ {
			if repeatCount > 1 {
				fmt.Printf(" [%d/%d]", j+1, repeatCount)
			}
			result := RunTask(ctx, client, task, ws)
			allResults = append(allResults, result)

			if result.IsError {
				fmt.Printf(" ERROR: %s\n", result.ErrorMsg)
			} else {
				fmt.Printf(" correctness=%.2f latency=%.1fs\n", result.Correctness, result.WallClockSeconds)
			}
		}
	}

	return allResults
}

// AggregateResults groups results by task and computes median/stddev for repeated runs.
func AggregateResults(results []RunResult) Summary {
	taskRuns := make(map[string][]RunResult)
	for _, r := range results {
		taskRuns[r.TaskID] = append(taskRuns[r.TaskID], r)
	}

	totalCorrectness := 0.0
	totalLatency := 0.0
	totalCost := 0.0
	taskCount := 0

	for _, runs := range taskRuns {
		if len(runs) == 0 {
			continue
		}
		taskCount++

		correctnessVals := make([]float64, len(runs))
		latencyVals := make([]float64, len(runs))
		for i, r := range runs {
			correctnessVals[i] = r.Correctness
			latencyVals[i] = r.WallClockSeconds
			totalCost += r.CostUSD
		}

		totalCorrectness += median(correctnessVals)
		totalLatency += median(latencyVals)
	}

	repeatCount := 1
	if taskCount > 0 && len(results) > taskCount {
		repeatCount = len(results) / taskCount
	}

	totalTokens := 0
	for _, r := range results {
		totalTokens += r.TotalTokens
	}

	summary := Summary{
		TotalTasks:  taskCount,
		TotalRuns:   len(results),
		RepeatCount: repeatCount,
		TotalCost:   totalCost,
		TotalTokens: totalTokens,
	}

	if taskCount > 0 {
		summary.AvgCorrectness = totalCorrectness / float64(taskCount)
		summary.AvgLatency = totalLatency / float64(taskCount)
	}

	// Efficiency metrics
	if totalTokens > 0 {
		summary.ScorePerKTokens = summary.AvgCorrectness / (float64(totalTokens) / 1000.0)
	}
	if totalCost > 0 {
		summary.ScorePerDollar = summary.AvgCorrectness / totalCost
	}

	return summary
}

func median(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sorted := make([]float64, len(vals))
	copy(sorted, vals)
	sort.Float64s(sorted)
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// Stddev computes standard deviation for a slice of float64.
func Stddev(vals []float64) float64 {
	if len(vals) <= 1 {
		return 0
	}
	mean := 0.0
	for _, v := range vals {
		mean += v
	}
	mean /= float64(len(vals))

	variance := 0.0
	for _, v := range vals {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(vals) - 1)
	return math.Sqrt(variance)
}

func generateRunID() string {
	b := make([]byte, 4)
	rand.Read(b)
	return fmt.Sprintf("%s_%x", time.Now().Format("20060102_150405"), b)
}
