package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"time"
)

const version = "0.6.0"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "run":
		cmdRun(os.Args[2:])
	case "compare":
		cmdCompare(os.Args[2:])
	case "list":
		cmdList()
	case "version":
		fmt.Printf("clawbench v%s\n", version)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", os.Args[1])
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`clawbench v%s — Benchmark your OpenClaw setup

Usage:
  clawbench run [flags]        Run benchmark tasks against your Gateway
  clawbench compare A.json B.json  Compare two benchmark result files
  clawbench list               List available benchmark tasks
  clawbench version            Print version

Run flags:
  --mode MODE       Backend mode: "cli" (default) or "websocket"
                    cli: uses openclaw CLI (handles auth automatically)
                    websocket: connects directly to Gateway WebSocket
  --gateway URL     Gateway WebSocket URL (default: ws://127.0.0.1:18789, websocket mode only)
  --token TOKEN     Auth token (websocket mode only, or set OPENCLAW_AUTH_TOKEN)
  --label NAME      Label for this benchmark run (default: timestamp)
  --benchmark NAME  Benchmark suite: "builtin" (default) or "exercism"
  --task ID         Run a specific task only (default: all)
  --repeat N        Repeat each task N times (default: 1)
  --workspace PATH  Path to OpenClaw workspace (for file_exists checks)
  --output PATH     Output JSON file (default: results/<label>.json)

Examples:
  clawbench run --label "my-setup-v2"
  clawbench run --benchmark exercism --label "my-setup"
  clawbench run --benchmark exercism --task exercism/two-fer
  clawbench compare results/setup-a.json results/setup-b.json
`, version)
}

func cmdRun(args []string) {
	// Parse flags
	mode := "cli" // default to CLI backend
	debug := false
	benchmark := "" // "", "builtin", "exercism"
	gatewayURL := "ws://127.0.0.1:18789"
	authToken := os.Getenv("OPENCLAW_AUTH_TOKEN")
	label := time.Now().Format("20060102-150405")
	taskID := ""
	repeatCount := 1
	workspacePath := ""
	outputPath := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--mode":
			i++
			if i < len(args) {
				mode = args[i]
			}
		case "--gateway":
			i++
			if i < len(args) {
				gatewayURL = args[i]
			}
		case "--token":
			i++
			if i < len(args) {
				authToken = args[i]
			}
		case "--label":
			i++
			if i < len(args) {
				label = args[i]
			}
		case "--task":
			i++
			if i < len(args) {
				taskID = args[i]
			}
		case "--repeat":
			i++
			if i < len(args) {
				fmt.Sscanf(args[i], "%d", &repeatCount)
			}
		case "--workspace":
			i++
			if i < len(args) {
				workspacePath = args[i]
			}
		case "--output":
			i++
			if i < len(args) {
				outputPath = args[i]
			}
		case "--debug":
			debug = true
		case "--benchmark":
			i++
			if i < len(args) {
				benchmark = args[i]
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown flag: %s\n", args[i])
			os.Exit(1)
		}
	}

	if outputPath == "" {
		os.MkdirAll("results", 0755)
		outputPath = fmt.Sprintf("results/%s.json", label)
	}

	// Select tasks based on benchmark suite
	var tasks []Task
	var exercismBench *ExercismBenchmark
	var taskWorkspaces map[string]string // taskID -> workspace path for exercism

	switch benchmark {
	case "exercism":
		home, _ := os.UserHomeDir()
		cacheDir := filepath.Join(home, ".clawbench", "exercism")
		os.MkdirAll(cacheDir, 0755)
		exercismBench = NewExercismBenchmark(cacheDir)
		if err := exercismBench.EnsureDownloaded(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %s\n", err)
			os.Exit(1)
		}
		var err error
		tasks, err = exercismBench.GenerateAllTasks()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading exercism tasks: %s\n", err)
			os.Exit(1)
		}
		// Build workspace map — each exercise runs in its own directory
		taskWorkspaces = make(map[string]string)
		for _, t := range tasks {
			name := strings.TrimPrefix(t.ID, "exercism/")
			taskWorkspaces[t.ID] = exercismBench.WorkspaceDir(name)
		}
		fmt.Printf("Loaded %d Exercism Python exercises\n", len(tasks))
	default:
		tasks = AllTasks()
	}

	if taskID != "" {
		var filtered []Task
		for _, t := range tasks {
			if t.ID == taskID {
				filtered = append(filtered, t)
				break
			}
		}
		if len(filtered) == 0 {
			fmt.Fprintf(os.Stderr, "unknown task: %s\nRun 'clawbench list' to see available tasks.\n", taskID)
			os.Exit(1)
		}
		tasks = filtered
	}

	// Set up context with interrupt handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)
	go func() {
		<-sigCh
		fmt.Println("\nInterrupted. Saving partial results...")
		cancel()
	}()

	// Create backend
	var client Backend
	switch mode {
	case "cli":
		fmt.Println("Using openclaw CLI backend...")
		client = NewCLIBackend()
	case "websocket":
		fmt.Printf("Connecting to Gateway at %s...\n", gatewayURL)
		gwClient := NewGatewayClient(gatewayURL, authToken)
		gwClient.debug = debug
		client = gwClient
	default:
		fmt.Fprintf(os.Stderr, "Unknown mode: %s (use 'cli' or 'websocket')\n", mode)
		os.Exit(1)
	}

	if err := client.Connect(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
	defer client.Close()

	gwVersion := client.ServerVersion()
	fmt.Printf("Connected. Gateway version: %s\n", gwVersion)
	fmt.Printf("Running %d task(s), %d repeat(s) each\n\n", len(tasks), repeatCount)

	// Run tasks
	results := RunAll(ctx, client, tasks, repeatCount, workspacePath, taskWorkspaces)

	// Build config metadata from first successful result
	config := ConfigMeta{
		Label:      label,
		GatewayURL: gatewayURL,
		GatewayVer: gwVersion,
	}
	for _, r := range results {
		if r.Config.Model != "" {
			config.Model = r.Config.Model
			config.Temperature = r.Config.Temperature
			break
		}
	}

	// Assemble full results
	benchResults := BenchmarkResults{
		Version:   version,
		Timestamp: time.Now(),
		Label:     label,
		Config:    config,
		Results:   results,
		Summary:   AggregateResults(results),
	}

	// Print and save
	PrintResults(benchResults)
	if err := SaveResults(benchResults, outputPath); err != nil {
		fmt.Fprintf(os.Stderr, "Error saving results: %s\n", err)
		os.Exit(1)
	}
}

func cmdCompare(args []string) {
	if len(args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: clawbench compare <file_a.json> <file_b.json>\n")
		os.Exit(1)
	}

	a, err := LoadResults(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %s\n", args[0], err)
		os.Exit(1)
	}
	b, err := LoadResults(args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading %s: %s\n", args[1], err)
		os.Exit(1)
	}

	CompareResults(a, b)
}

func cmdList() {
	tasks := AllTasks()
	fmt.Printf("Available benchmark tasks (%d):\n\n", len(tasks))
	for _, t := range tasks {
		fmt.Printf("  %-25s  %s\n", t.ID, t.Name)
		fmt.Printf("  %-25s  category: %s, budget: %s\n", "", t.Category, t.TimeBudget)
		fmt.Printf("  %-25s  evaluators: ", "")
		var evalTypes []string
		for _, e := range t.Evaluators {
			evalTypes = append(evalTypes, e.Type)
		}
		fmt.Printf("%s\n\n", joinUnique(evalTypes))
	}
}

func joinUnique(strs []string) string {
	seen := make(map[string]bool)
	var unique []string
	for _, s := range strs {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}
	return fmt.Sprintf("%s", unique)
}
