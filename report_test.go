package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestSaveAndLoadResults(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test-results.json")

	original := BenchmarkResults{
		Version:   "0.1.0",
		Timestamp: time.Now().Truncate(time.Second),
		Label:     "test-run",
		Config: ConfigMeta{
			Label:      "test-run",
			Model:      "claude-sonnet-4-6",
			GatewayURL: "ws://127.0.0.1:18789",
		},
		Results: []RunResult{
			{
				RunID:            "20260331_120000_abcd1234",
				TaskID:           "skill_tool_chain",
				Timestamp:        time.Now().Truncate(time.Second),
				Correctness:      0.75,
				ToolAccuracy:     1.0,
				WallClockSeconds: 3.5,
				CostUSD:          0.003,
				ToolsUsed:        []string{"weather", "file_write"},
				EvalResults: []EvalResult{
					{Type: "tool_invoked", Score: 1.0, Weight: 1.0, Passed: true, Details: "ok"},
				},
				RawResponse: "test response",
			},
		},
		Summary: Summary{
			TotalTasks:     1,
			TotalRuns:      1,
			RepeatCount:    1,
			AvgCorrectness: 0.75,
			AvgLatency:     3.5,
			TotalCost:      0.003,
		},
	}

	// Save
	if err := SaveResults(original, path); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	// Load
	loaded, err := LoadResults(path)
	if err != nil {
		t.Fatalf("load failed: %v", err)
	}

	// Verify
	if loaded.Label != original.Label {
		t.Errorf("label mismatch: %s vs %s", loaded.Label, original.Label)
	}
	if loaded.Config.Model != original.Config.Model {
		t.Errorf("model mismatch: %s vs %s", loaded.Config.Model, original.Config.Model)
	}
	if len(loaded.Results) != len(original.Results) {
		t.Fatalf("results count mismatch: %d vs %d", len(loaded.Results), len(original.Results))
	}
	if loaded.Results[0].Correctness != original.Results[0].Correctness {
		t.Errorf("correctness mismatch: %f vs %f",
			loaded.Results[0].Correctness, original.Results[0].Correctness)
	}
}

func TestLoadResults_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := LoadResults(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadResults_MissingFile(t *testing.T) {
	_, err := LoadResults("/nonexistent/path/results.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestAggregateResults(t *testing.T) {
	results := []RunResult{
		{TaskID: "task_a", Correctness: 0.8, WallClockSeconds: 2.0, CostUSD: 0.001},
		{TaskID: "task_a", Correctness: 1.0, WallClockSeconds: 3.0, CostUSD: 0.002},
		{TaskID: "task_a", Correctness: 0.6, WallClockSeconds: 2.5, CostUSD: 0.001},
		{TaskID: "task_b", Correctness: 0.5, WallClockSeconds: 5.0, CostUSD: 0.005},
	}

	summary := AggregateResults(results)

	if summary.TotalTasks != 2 {
		t.Errorf("expected 2 tasks, got %d", summary.TotalTasks)
	}
	if summary.TotalRuns != 4 {
		t.Errorf("expected 4 runs, got %d", summary.TotalRuns)
	}
	// Median correctness for task_a: median(0.6, 0.8, 1.0) = 0.8
	// Median correctness for task_b: 0.5
	// Avg: (0.8 + 0.5) / 2 = 0.65
	if summary.AvgCorrectness < 0.64 || summary.AvgCorrectness > 0.66 {
		t.Errorf("expected avg correctness ~0.65, got %f", summary.AvgCorrectness)
	}
}

func TestMedian(t *testing.T) {
	tests := []struct {
		input    []float64
		expected float64
	}{
		{[]float64{1, 2, 3}, 2},
		{[]float64{1, 2, 3, 4}, 2.5},
		{[]float64{5}, 5},
		{nil, 0},
		{[]float64{3, 1, 2}, 2},
	}

	for _, tt := range tests {
		got := median(tt.input)
		if got != tt.expected {
			t.Errorf("median(%v) = %f, want %f", tt.input, got, tt.expected)
		}
	}
}

func TestStddev(t *testing.T) {
	vals := []float64{2, 4, 4, 4, 5, 5, 7, 9}
	sd := Stddev(vals)
	// Expected sample stddev ≈ 2.14
	if sd < 2.0 || sd > 2.3 {
		t.Errorf("expected stddev ~2.0, got %f", sd)
	}
}

func TestStddev_SingleValue(t *testing.T) {
	sd := Stddev([]float64{42})
	if sd != 0 {
		t.Errorf("expected stddev 0 for single value, got %f", sd)
	}
}
