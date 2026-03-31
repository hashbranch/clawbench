package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExactMatch_AllMatch(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: []string{`hello`, `world`}, Weight: 1.0}
	result := evalExactMatch(ec, "hello world")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
}

func TestExactMatch_PartialMatch(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: []string{`hello`, `missing`}, Weight: 1.0}
	result := evalExactMatch(ec, "hello world")
	if result.Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", result.Score)
	}
	if result.Passed {
		t.Error("expected passed=false for partial match")
	}
}

func TestExactMatch_NoMatch(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: []string{`missing`}, Weight: 1.0}
	result := evalExactMatch(ec, "hello world")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestExactMatch_EmptyResponse(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: []string{`hello`}, Weight: 1.0}
	result := evalExactMatch(ec, "")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestExactMatch_InvalidRegex(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: []string{`[invalid`}, Weight: 1.0}
	result := evalExactMatch(ec, "hello")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 for invalid regex, got %f", result.Score)
	}
	if result.Details == "" {
		t.Error("expected details about invalid regex")
	}
}

func TestExactMatch_NoPatterns(t *testing.T) {
	ec := EvalConfig{Type: "exact_match", Patterns: nil, Weight: 1.0}
	result := evalExactMatch(ec, "hello")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 with no patterns, got %f", result.Score)
	}
}

func TestToolInvoked_FromTrace(t *testing.T) {
	ec := EvalConfig{Type: "tool_invoked", ToolName: "weather", Weight: 1.0}
	resp := GatewayResponse{
		ToolCalls: []ToolCall{{Name: "weather"}},
	}
	result := evalToolInvoked(ec, resp)
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
}

func TestToolInvoked_FromTrace_CaseInsensitive(t *testing.T) {
	ec := EvalConfig{Type: "tool_invoked", ToolName: "Weather", Weight: 1.0}
	resp := GatewayResponse{
		ToolCalls: []ToolCall{{Name: "weather"}},
	}
	result := evalToolInvoked(ec, resp)
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for case-insensitive match, got %f", result.Score)
	}
}

func TestToolInvoked_NotFound(t *testing.T) {
	ec := EvalConfig{Type: "tool_invoked", ToolName: "weather", Weight: 1.0}
	resp := GatewayResponse{
		Text: "I don't have access to that tool.",
	}
	result := evalToolInvoked(ec, resp)
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestToolInvoked_HeuristicFallback(t *testing.T) {
	ec := EvalConfig{Type: "tool_invoked", ToolName: "weather", Weight: 1.0}
	resp := GatewayResponse{
		Text: "I used weather to check the forecast.",
	}
	result := evalToolInvoked(ec, resp)
	if result.Score != 0.5 {
		t.Errorf("expected score 0.5 for heuristic match, got %f", result.Score)
	}
}

func TestToolInvoked_NoToolData(t *testing.T) {
	ec := EvalConfig{Type: "tool_invoked", ToolName: "weather", Weight: 1.0}
	resp := GatewayResponse{
		Text: "The temperature in San Francisco is 65F.",
	}
	result := evalToolInvoked(ec, resp)
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 when tool not mentioned, got %f", result.Score)
	}
}

func TestFileExists_Present(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "test.txt"), []byte("hello"), 0644)

	ec := EvalConfig{Type: "file_exists", Path: "test.txt", Weight: 1.0}
	result := evalFileExists(ec, dir)
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
}

func TestFileExists_Absent(t *testing.T) {
	dir := t.TempDir()
	ec := EvalConfig{Type: "file_exists", Path: "missing.txt", Weight: 1.0}
	result := evalFileExists(ec, dir)
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestFileExists_NoWorkspace(t *testing.T) {
	ec := EvalConfig{Type: "file_exists", Path: "test.txt", Weight: 1.0}
	result := evalFileExists(ec, "")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
	if result.Details == "" {
		t.Error("expected details about missing workspace")
	}
}

func TestCost_WithTokens(t *testing.T) {
	ec := EvalConfig{Type: "cost", Weight: 0.3}
	resp := GatewayResponse{
		Model: "claude-sonnet-4-6",
		Tokens: TokenUsage{
			Input:     100,
			Output:    50,
			Total:     150,
			Available: true,
		},
	}
	result := evalCost(ec, resp)
	if result.Score <= 0 {
		t.Error("expected positive cost")
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
}

func TestCost_WithoutTokens(t *testing.T) {
	ec := EvalConfig{Type: "cost", Weight: 0.3}
	resp := GatewayResponse{
		Text:   "This is a response that is about 40 characters.",
		Tokens: TokenUsage{Available: false},
	}
	result := evalCost(ec, resp)
	if result.Score <= 0 {
		t.Error("expected positive estimated cost")
	}
	if result.Details == "" || result.Details[0] != '~' {
		t.Error("expected estimated cost indicator in details")
	}
}

func TestCost_LocalModel(t *testing.T) {
	ec := EvalConfig{Type: "cost", Weight: 0.3}
	resp := GatewayResponse{
		Model: "ollama/llama3",
		Tokens: TokenUsage{
			Input:     100,
			Output:    50,
			Total:     150,
			Available: true,
		},
	}
	result := evalCost(ec, resp)
	if result.Score != 0 {
		t.Errorf("expected 0 cost for local model, got %f", result.Score)
	}
}

func TestLatency(t *testing.T) {
	ec := EvalConfig{Type: "latency", Weight: 0.3}
	result := evalLatency(ec, 3.5)
	if result.Score != 3.5 {
		t.Errorf("expected score 3.5, got %f", result.Score)
	}
}

func TestFormatBullets_Perfect(t *testing.T) {
	ec := EvalConfig{Type: "format_bullets", Weight: 1.0}
	text := `- Renewable energy reduces carbon emissions significantly.
- Solar and wind power are increasingly cost-effective.
- Clean energy creates sustainable jobs worldwide.`

	result := evalFormatBullets(ec, text)
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f (%s)", result.Score, result.Details)
	}
}

func TestFormatBullets_WrongCount(t *testing.T) {
	ec := EvalConfig{Type: "format_bullets", Weight: 1.0}
	text := `- First point.
- Second point.`

	result := evalFormatBullets(ec, text)
	if result.Score >= 1.0 {
		t.Errorf("expected score < 1.0 for 2 bullets, got %f", result.Score)
	}
}

func TestFormatBullets_TooLong(t *testing.T) {
	ec := EvalConfig{Type: "format_bullets", Weight: 1.0}
	text := `- Renewable energy significantly reduces harmful carbon emissions while also providing sustainable and affordable long-term energy solutions for the entire global population and future generations to come.
- Solar power is good.
- Wind power is good.`

	result := evalFormatBullets(ec, text)
	if result.Score == 1.0 {
		t.Error("expected score < 1.0 when bullets exceed 20 words")
	}
}

func TestFormatBullets_NumberedList(t *testing.T) {
	ec := EvalConfig{Type: "format_bullets", Weight: 1.0}
	text := `1. Reduces carbon emissions effectively.
2. Solar and wind are cost-effective.
3. Creates sustainable energy jobs.`

	result := evalFormatBullets(ec, text)
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for numbered list, got %f (%s)", result.Score, result.Details)
	}
}

func TestComputeCorrectness(t *testing.T) {
	results := []EvalResult{
		{Type: "exact_match", Score: 0.5, Weight: 1.0},
		{Type: "format_bullets", Score: 1.0, Weight: 1.0},
		{Type: "cost", Score: 0.001, Weight: 0.3},     // should be excluded
		{Type: "latency", Score: 2.0, Weight: 0.3},     // should be excluded
	}
	corr := ComputeCorrectness(results)
	expected := (0.5*1.0 + 1.0*1.0) / (1.0 + 1.0) // 0.75
	if corr != expected {
		t.Errorf("expected %.2f, got %.2f", expected, corr)
	}
}

func TestComputeToolAccuracy(t *testing.T) {
	results := []EvalResult{
		{Type: "tool_invoked", Score: 1.0, Weight: 1.0},
		{Type: "tool_invoked", Score: 0.0, Weight: 1.0},
		{Type: "file_exists", Score: 1.0, Weight: 1.0},
		{Type: "exact_match", Score: 0.5, Weight: 1.0}, // should be excluded
	}
	acc := ComputeToolAccuracy(results)
	// expected: (1.0 + 0.0 + 1.0) / 3.0 = 0.666...
	if acc < 0.66 || acc > 0.67 {
		t.Errorf("expected ~0.67, got %.2f", acc)
	}
}
