package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
)

// PrintResults displays run results as a formatted terminal table.
func PrintResults(results BenchmarkResults) {
	fmt.Printf("\n=== ClawBench Results: %s ===\n", results.Label)
	fmt.Printf("Gateway: %s | Model: %s", results.Config.GatewayURL, results.Config.Model)
	if results.Config.Temperature > 0 {
		fmt.Printf(" | Temp: %.1f", results.Config.Temperature)
	}
	fmt.Println()
	fmt.Println(strings.Repeat("-", 70))

	for _, r := range results.Results {
		status := "PASS"
		if r.IsError {
			status = "ERROR"
		}

		fmt.Printf("\n  %s [%s]\n", r.TaskID, status)
		if r.IsError {
			fmt.Printf("    error: %s\n", r.ErrorMsg)
			continue
		}

		fmt.Printf("    correctness:    %.2f\n", r.Correctness)
		if r.ToolAccuracy > 0 {
			fmt.Printf("    tool accuracy:  %.2f\n", r.ToolAccuracy)
		}
		fmt.Printf("    latency:        %.2fs\n", r.WallClockSeconds)
		fmt.Printf("    cost:           $%.6f\n", r.CostUSD)
		if len(r.ToolsUsed) > 0 {
			fmt.Printf("    tools used:     %s\n", strings.Join(r.ToolsUsed, ", "))
		}

		// Show evaluator details
		for _, er := range r.EvalResults {
			icon := "  "
			if er.Passed {
				icon = "OK"
			}
			fmt.Printf("      [%s] %s: %s\n", icon, er.Type, er.Details)
		}
	}

	fmt.Println(strings.Repeat("-", 70))
	fmt.Printf("  Summary: %d tasks, %d runs", results.Summary.TotalTasks, results.Summary.TotalRuns)
	if results.Summary.RepeatCount > 1 {
		fmt.Printf(" (%dx repeat, using medians)", results.Summary.RepeatCount)
	}
	fmt.Println()
	fmt.Printf("  Avg correctness: %.2f | Avg latency: %.2fs | Total cost: $%.6f\n",
		results.Summary.AvgCorrectness, results.Summary.AvgLatency, results.Summary.TotalCost)
	if results.Summary.ScorePerKTokens > 0 {
		fmt.Printf("  Efficiency: %.4f score/1K tokens | %.2f score/$\n",
			results.Summary.ScorePerKTokens, results.Summary.ScorePerDollar)
	}
	fmt.Println()
}

// SaveResults writes benchmark results to a JSON file.
func SaveResults(results BenchmarkResults, path string) error {
	data, err := json.MarshalIndent(results, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal results: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write results to %s: %w", path, err)
	}
	fmt.Printf("Results saved to: %s\n", path)
	return nil
}

// LoadResults reads benchmark results from a JSON file.
func LoadResults(path string) (BenchmarkResults, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return BenchmarkResults{}, fmt.Errorf("failed to read %s: %w", path, err)
	}
	var results BenchmarkResults
	if err := json.Unmarshal(data, &results); err != nil {
		return BenchmarkResults{}, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return results, nil
}

// CompareResults prints a side-by-side comparison of two benchmark result sets.
func CompareResults(a, b BenchmarkResults) {
	fmt.Printf("\n=== ClawBench Compare ===\n")
	fmt.Printf("  A: %s (model: %s)\n", a.Label, a.Config.Model)
	fmt.Printf("  B: %s (model: %s)\n", b.Label, b.Config.Model)

	// Temperature mismatch warning
	if a.Config.Temperature != b.Config.Temperature &&
		(a.Config.Temperature > 0 || b.Config.Temperature > 0) {
		fmt.Printf("\n  WARNING: temperature mismatch (A=%.1f, B=%.1f) — results may not be directly comparable\n",
			a.Config.Temperature, b.Config.Temperature)
	}

	fmt.Println()
	// Header
	fmt.Printf("  %-24s  %10s  %10s  %10s\n", "Metric", a.Label, b.Label, "Delta")
	fmt.Printf("  %s\n", strings.Repeat("-", 60))

	// Build per-task maps
	aByTask := mapByTask(a.Results)
	bByTask := mapByTask(b.Results)

	// Collect all task IDs
	taskIDs := make(map[string]bool)
	for id := range aByTask {
		taskIDs[id] = true
	}
	for id := range bByTask {
		taskIDs[id] = true
	}

	for id := range taskIDs {
		aRuns := aByTask[id]
		bRuns := bByTask[id]

		fmt.Printf("\n  %s\n", id)

		if len(aRuns) == 0 {
			fmt.Printf("    (no results in A)\n")
			continue
		}
		if len(bRuns) == 0 {
			fmt.Printf("    (no results in B)\n")
			continue
		}

		// Use median values for comparison
		aCorr := medianField(aRuns, func(r RunResult) float64 { return r.Correctness })
		bCorr := medianField(bRuns, func(r RunResult) float64 { return r.Correctness })
		printCompareRow("correctness", aCorr, bCorr, "%.2f", true)

		aTool := medianField(aRuns, func(r RunResult) float64 { return r.ToolAccuracy })
		bTool := medianField(bRuns, func(r RunResult) float64 { return r.ToolAccuracy })
		if aTool > 0 || bTool > 0 {
			printCompareRow("tool accuracy", aTool, bTool, "%.2f", true)
		}

		aLat := medianField(aRuns, func(r RunResult) float64 { return r.WallClockSeconds })
		bLat := medianField(bRuns, func(r RunResult) float64 { return r.WallClockSeconds })
		printCompareRow("latency (s)", aLat, bLat, "%.2f", false) // lower is better

		aCost := medianField(aRuns, func(r RunResult) float64 { return r.CostUSD })
		bCost := medianField(bRuns, func(r RunResult) float64 { return r.CostUSD })
		printCompareRow("cost ($)", aCost, bCost, "%.6f", false) // lower is better
	}

	// Overall summary
	fmt.Printf("\n  %s\n", strings.Repeat("-", 60))
	fmt.Printf("  SUMMARY\n")
	printCompareRow("avg correctness", a.Summary.AvgCorrectness, b.Summary.AvgCorrectness, "%.2f", true)
	printCompareRow("avg latency", a.Summary.AvgLatency, b.Summary.AvgLatency, "%.2f", false)
	printCompareRow("total cost", a.Summary.TotalCost, b.Summary.TotalCost, "%.6f", false)
	fmt.Println()
}

func printCompareRow(label string, aVal, bVal float64, format string, higherIsBetter bool) {
	aStr := fmt.Sprintf(format, aVal)
	bStr := fmt.Sprintf(format, bVal)

	delta := ""
	if aVal != 0 {
		pct := (bVal - aVal) / math.Abs(aVal) * 100
		arrow := ""
		if pct > 0.5 {
			if higherIsBetter {
				arrow = " ^" // up is good
			} else {
				arrow = " v" // up is bad for latency/cost
			}
		} else if pct < -0.5 {
			if higherIsBetter {
				arrow = " v" // down is bad
			} else {
				arrow = " ^" // down is good for latency/cost
			}
		}
		delta = fmt.Sprintf("%+.0f%%%s", pct, arrow)
	}

	fmt.Printf("    %-22s  %10s  %10s  %10s\n", label, aStr, bStr, delta)
}

func mapByTask(results []RunResult) map[string][]RunResult {
	m := make(map[string][]RunResult)
	for _, r := range results {
		m[r.TaskID] = append(m[r.TaskID], r)
	}
	return m
}

func medianField(runs []RunResult, field func(RunResult) float64) float64 {
	vals := make([]float64, len(runs))
	for i, r := range runs {
		vals[i] = field(r)
	}
	return median(vals)
}
