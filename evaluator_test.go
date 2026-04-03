package main

import (
	"os"
	"path/filepath"
	"strings"
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

// --- regex_reject evaluator tests ---

func TestRegexReject_NoViolations(t *testing.T) {
	ec := EvalConfig{Type: "regex_reject", Patterns: []string{`badword`, `terrible`}, Weight: 1.0}
	result := evalRegexReject(ec, "this is a clean response")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f", result.Score)
	}
	if !result.Passed {
		t.Error("expected passed=true")
	}
}

func TestRegexReject_OneViolation(t *testing.T) {
	ec := EvalConfig{Type: "regex_reject", Patterns: []string{`badword`, `clean`}, Weight: 1.0}
	result := evalRegexReject(ec, "this is a clean response")
	if result.Score != 0.5 {
		t.Errorf("expected score 0.5, got %f", result.Score)
	}
	if result.Passed {
		t.Error("expected passed=false")
	}
}

func TestRegexReject_AllViolations(t *testing.T) {
	ec := EvalConfig{Type: "regex_reject", Patterns: []string{`this`, `response`}, Weight: 1.0}
	result := evalRegexReject(ec, "this is a response")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestRegexReject_EmptyPatterns(t *testing.T) {
	ec := EvalConfig{Type: "regex_reject", Patterns: nil, Weight: 1.0}
	result := evalRegexReject(ec, "anything")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 with no patterns, got %f", result.Score)
	}
}

// --- response_check evaluator tests ---

func TestResponseCheck_PassingScript(t *testing.T) {
	ec := EvalConfig{Type: "response_check", Path: "grep -q hello", Weight: 1.0}
	result := evalResponseCheck(ec, "hello world", t.TempDir())
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f (%s)", result.Score, result.Details)
	}
}

func TestResponseCheck_FailingScript(t *testing.T) {
	ec := EvalConfig{Type: "response_check", Path: "grep -q missing", Weight: 1.0}
	result := evalResponseCheck(ec, "hello world", t.TempDir())
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0, got %f", result.Score)
	}
}

func TestResponseCheck_NoCommand(t *testing.T) {
	ec := EvalConfig{Type: "response_check", Path: "", Weight: 1.0}
	result := evalResponseCheck(ec, "hello", t.TempDir())
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 with no command, got %f", result.Score)
	}
}

// --- GAIA Exact evaluator tests ---

func TestGAIAExact_StringMatch(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"Canberra"}, Weight: 1.0}
	result := evalGAIAExact(ec, "Canberra")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_CaseInsensitive(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"Canberra"}, Weight: 1.0}
	result := evalGAIAExact(ec, "CANBERRA")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for case-insensitive match, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_WhitespaceNormalization(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"Neil Armstrong"}, Weight: 1.0}
	result := evalGAIAExact(ec, "  Neil   Armstrong  ")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 with whitespace normalization, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_PunctuationRemoval(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"sea gull"}, Weight: 1.0}
	result := evalGAIAExact(ec, "sea-gull.")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 with punctuation normalization, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_NumericMatch(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"42"}, Weight: 1.0}
	result := evalGAIAExact(ec, "42")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for numeric match, got %f", result.Score)
	}
}

func TestGAIAExact_NumericWithCommas(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"1000000"}, Weight: 1.0}
	result := evalGAIAExact(ec, "1,000,000")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for numeric match with commas, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_NumericWithDollarSign(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"38.79"}, Weight: 1.0}
	result := evalGAIAExact(ec, "$38.79")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for numeric match with dollar sign, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_ListMatch(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"Neil Armstrong, 1969"}, Weight: 1.0}
	result := evalGAIAExact(ec, "Neil Armstrong, 1969")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 for list match, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_ListDifferentLength(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"a, b, c"}, Weight: 1.0}
	result := evalGAIAExact(ec, "a, b")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 for list length mismatch, got %f", result.Score)
	}
}

func TestGAIAExact_Mismatch(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"Paris"}, Weight: 1.0}
	result := evalGAIAExact(ec, "London")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 for mismatch, got %f", result.Score)
	}
}

func TestGAIAExact_ExtractAnswer(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"42"}, Weight: 1.0}
	result := evalGAIAExact(ec, "After careful analysis, the answer is: 42")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 when extracting from verbose response, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_FinalAnswerExtraction(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: []string{"255"}, Weight: 1.0}
	result := evalGAIAExact(ec, "Let me calculate...\n\nFINAL ANSWER: 255")
	if result.Score != 1.0 {
		t.Errorf("expected score 1.0 extracting FINAL ANSWER, got %f (%s)", result.Score, result.Details)
	}
}

func TestGAIAExact_NoPatterns(t *testing.T) {
	ec := EvalConfig{Type: "gaia_exact", Patterns: nil, Weight: 1.0}
	result := evalGAIAExact(ec, "hello")
	if result.Score != 0.0 {
		t.Errorf("expected score 0.0 with no patterns, got %f", result.Score)
	}
}

func TestExtractGAIAAnswer_LastLine(t *testing.T) {
	answer := extractGAIAAnswer("Some reasoning...\n\nLet me think about this.\n\n42")
	if answer != "42" {
		t.Errorf("expected '42', got %q", answer)
	}
}

func TestGaiaStrNormalize(t *testing.T) {
	tests := []struct {
		input, expected string
	}{
		{"Hello World!", "helloworld"},
		{"  Neil  Armstrong  ", "neilarmstrong"},
		{"sea-gull.", "seagull"},
		{"Structured Query Language", "structuredquerylanguage"},
	}
	for _, tc := range tests {
		got := gaiaStrNormalize(tc.input)
		if got != tc.expected {
			t.Errorf("gaiaStrNormalize(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestClawBenchOriginalTasks_Count(t *testing.T) {
	tasks := ClawBenchOriginalTasks()
	if len(tasks) != 15 {
		t.Errorf("expected 15 ClawBench original tasks, got %d", len(tasks))
	}
	// Verify all have clawbench_original category and cb_ prefix
	for _, task := range tasks {
		if task.Category != "clawbench_original" {
			t.Errorf("task %s has category %q, expected clawbench_original", task.ID, task.Category)
		}
		if !strings.HasPrefix(task.ID, "cb_reasoning_") {
			t.Errorf("task %s should have cb_reasoning_ prefix", task.ID)
		}
	}
}

func TestAllTasks_IncludesOriginals(t *testing.T) {
	all := AllTasks()
	builtin := BuiltinTasks()
	originals := ClawBenchOriginalTasks()
	regression := RegressionTasks()
	expected := len(builtin) + len(originals) + len(regression)
	if len(all) != expected {
		t.Errorf("AllTasks() returned %d tasks, expected %d+%d+%d=%d", len(all), len(builtin), len(originals), len(regression), expected)
	}
}

func TestRegressionTasks_Count(t *testing.T) {
	tasks := RegressionTasks()
	if len(tasks) != 3 {
		t.Errorf("expected 3 regression tasks, got %d", len(tasks))
	}
	for _, task := range tasks {
		if task.Category != "regression" {
			t.Errorf("task %s has category %q, expected regression", task.ID, task.Category)
		}
		if !strings.HasPrefix(task.ID, "reg_") {
			t.Errorf("task %s should have reg_ prefix", task.ID)
		}
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
