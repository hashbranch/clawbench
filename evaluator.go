package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// Evaluate runs all evaluators for a task against a GatewayResponse.
// workspacePath is needed for file_exists checks.
func Evaluate(task Task, resp GatewayResponse, wallClock float64, workspacePath string) []EvalResult {
	var results []EvalResult
	for _, ec := range task.Evaluators {
		result := runEvaluator(ec, resp, wallClock, workspacePath)
		results = append(results, result)
	}
	return results
}

func runEvaluator(ec EvalConfig, resp GatewayResponse, wallClock float64, workspacePath string) EvalResult {
	switch ec.Type {
	case "exact_match":
		return evalExactMatch(ec, resp.Text)
	case "gaia_exact":
		return evalGAIAExact(ec, resp.Text)
	case "tool_invoked":
		return evalToolInvoked(ec, resp)
	case "file_exists":
		return evalFileExists(ec, workspacePath)
	case "cost":
		return evalCost(ec, resp)
	case "latency":
		return evalLatency(ec, wallClock)
	case "format_bullets":
		return evalFormatBullets(ec, resp.Text)
	case "exec_check":
		return evalExecCheck(ec, workspacePath)
	case "regex_reject":
		return evalRegexReject(ec, resp.Text)
	case "response_check":
		return evalResponseCheck(ec, resp.Text, workspacePath)
	default:
		return EvalResult{
			Type:    ec.Type,
			Score:   0,
			Weight:  ec.Weight,
			Passed:  false,
			Details: fmt.Sprintf("unknown evaluator type: %s", ec.Type),
		}
	}
}

// evalGAIAExact implements the GAIA benchmark's exact-match scoring.
// It normalizes both the model answer and ground truth by:
// - Lowercasing
// - Removing all whitespace
// - Removing punctuation
// - For numbers: comparing numeric values (handles $, %, commas)
// - For comma/semicolon-separated lists: comparing element-by-element
//
// This matches the official GAIA scorer from the benchmark paper.
func evalGAIAExact(ec EvalConfig, text string) EvalResult {
	if len(ec.Patterns) == 0 {
		return EvalResult{Type: "gaia_exact", Score: 0, Weight: ec.Weight, Details: "no ground truth specified"}
	}
	groundTruth := ec.Patterns[0] // GAIA uses a single ground truth answer

	// Extract the final answer from the response text.
	// Agents often wrap answers in various formats; try to extract the core answer.
	answer := extractGAIAAnswer(text)

	matched := gaiaQuestionScorer(answer, groundTruth)

	score := 0.0
	if matched {
		score = 1.0
	}
	return EvalResult{
		Type:    "gaia_exact",
		Score:   score,
		Weight:  ec.Weight,
		Passed:  matched,
		Details: fmt.Sprintf("expected=%q, extracted=%q, matched=%v", groundTruth, answer, matched),
	}
}

// gaiaQuestionScorer implements the official GAIA scoring logic.
func gaiaQuestionScorer(modelAnswer, groundTruth string) bool {
	// If ground truth is a number, compare numerically
	if isFloat(groundTruth) {
		normalizedAnswer := normalizeNumberStr(modelAnswer)
		gt, _ := parseFloat(groundTruth)
		return normalizedAnswer == gt
	}

	// If ground truth contains commas or semicolons, compare as list
	if strings.ContainsAny(groundTruth, ",;") {
		return gaiaListCompare(modelAnswer, groundTruth)
	}

	// Otherwise compare as normalized strings
	return gaiaStrNormalize(modelAnswer) == gaiaStrNormalize(groundTruth)
}

// gaiaListCompare compares comma/semicolon-separated lists element-by-element.
func gaiaListCompare(modelAnswer, groundTruth string) bool {
	gtElems := splitOnAny(groundTruth, ",;")
	maElems := splitOnAny(modelAnswer, ",;")
	if len(gtElems) != len(maElems) {
		return false
	}
	for i, gt := range gtElems {
		ma := maElems[i]
		if isFloat(strings.TrimSpace(gt)) {
			if normalizeNumberStr(ma) != normalizeNumberStr(gt) {
				return false
			}
		} else {
			if gaiaStrNormalize(ma) != gaiaStrNormalize(gt) {
				return false
			}
		}
	}
	return true
}

// gaiaStrNormalize removes whitespace, punctuation, and lowercases the string.
func gaiaStrNormalize(s string) string {
	s = strings.ToLower(s)
	// Remove all whitespace
	s = strings.Join(strings.Fields(s), "")
	// Remove punctuation
	var b strings.Builder
	for _, r := range s {
		if !strings.ContainsRune("!\"#$%&'()*+,-./:;<=>?@[\\]^_`{|}~", r) {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// normalizeNumberStr strips common formatting ($, %, commas) and parses as float.
func normalizeNumberStr(s string) float64 {
	s = strings.TrimSpace(s)
	for _, ch := range []string{"$", "%", ","} {
		s = strings.Replace(s, ch, "", -1)
	}
	f, err := parseFloat(s)
	if err != nil {
		return -999999.999 // sentinel for non-parseable
	}
	return f
}

func isFloat(s string) bool {
	s = strings.TrimSpace(s)
	for _, ch := range []string{"$", "%", ","} {
		s = strings.Replace(s, ch, "", -1)
	}
	_, err := parseFloat(s)
	return err == nil
}

func parseFloat(s string) (float64, error) {
	s = strings.TrimSpace(s)
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func splitOnAny(s string, chars string) []string {
	f := func(r rune) bool { return strings.ContainsRune(chars, r) }
	return strings.FieldsFunc(s, f)
}

// extractGAIAAnswer tries to extract a concise final answer from agent output.
// Agents often produce verbose responses; we look for common answer patterns.
func extractGAIAAnswer(text string) string {
	text = strings.TrimSpace(text)
	if text == "" {
		return ""
	}

	// Look for explicit "FINAL ANSWER:" or "The answer is:" patterns (case-insensitive)
	patterns := []string{
		`(?i)(?:final answer|the answer is|answer:)\s*:?\s*(.+?)(?:\n|$)`,
		`(?i)(?:the result is|result:)\s*:?\s*(.+?)(?:\n|$)`,
	}
	for _, p := range patterns {
		re := regexp.MustCompile(p)
		if m := re.FindStringSubmatch(text); len(m) > 1 {
			ans := strings.TrimSpace(m[1])
			// Strip leading colon/punctuation that might be captured
			ans = strings.TrimLeft(ans, ": ")
			if ans != "" {
				return ans
			}
		}
	}

	// If the response is short (likely just the answer), use it directly
	lines := strings.Split(text, "\n")
	// Use the last non-empty line as the answer (agents often put answer last)
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" {
			return line
		}
	}

	return text
}

// evalExactMatch checks if the response text matches any of the regex patterns.
func evalExactMatch(ec EvalConfig, text string) EvalResult {
	if len(ec.Patterns) == 0 {
		return EvalResult{Type: "exact_match", Score: 0, Weight: ec.Weight, Details: "no patterns specified"}
	}

	matched := 0
	var errors []string
	for _, pattern := range ec.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			errors = append(errors, fmt.Sprintf("invalid regex %q: %v", pattern, err))
			continue
		}
		if re.MatchString(text) {
			matched++
		}
	}

	score := float64(matched) / float64(len(ec.Patterns))
	details := fmt.Sprintf("%d/%d patterns matched", matched, len(ec.Patterns))
	if len(errors) > 0 {
		details += "; " + strings.Join(errors, "; ")
	}

	return EvalResult{
		Type:    "exact_match",
		Score:   score,
		Weight:  ec.Weight,
		Passed:  matched == len(ec.Patterns),
		Details: details,
	}
}

// evalToolInvoked checks if a specific tool was used in the response.
func evalToolInvoked(ec EvalConfig, resp GatewayResponse) EvalResult {
	// First: check structured tool call data from Gateway
	for _, tc := range resp.ToolCalls {
		if strings.EqualFold(tc.Name, ec.ToolName) {
			return EvalResult{
				Type:    "tool_invoked",
				Score:   1.0,
				Weight:  ec.Weight,
				Passed:  true,
				Details: fmt.Sprintf("tool %q invoked (from trace)", ec.ToolName),
			}
		}
	}

	// Fallback: check response text for heuristic evidence
	// This handles the case where Gateway doesn't expose tool call details
	heuristics := []string{
		ec.ToolName,
		"used " + ec.ToolName,
		"called " + ec.ToolName,
		"using " + ec.ToolName,
	}
	lower := strings.ToLower(resp.Text)
	for _, h := range heuristics {
		if strings.Contains(lower, strings.ToLower(h)) {
			return EvalResult{
				Type:    "tool_invoked",
				Score:   0.5, // lower confidence for heuristic match
				Weight:  ec.Weight,
				Passed:  true,
				Details: fmt.Sprintf("tool %q likely invoked (heuristic match in response text)", ec.ToolName),
			}
		}
	}

	return EvalResult{
		Type:    "tool_invoked",
		Score:   0,
		Weight:  ec.Weight,
		Passed:  false,
		Details: fmt.Sprintf("tool %q not detected", ec.ToolName),
	}
}

// evalFileExists checks if an expected output file was created.
func evalFileExists(ec EvalConfig, workspacePath string) EvalResult {
	if workspacePath == "" {
		return EvalResult{
			Type:    "file_exists",
			Score:   0,
			Weight:  ec.Weight,
			Passed:  false,
			Details: "no workspace path specified (use --workspace flag)",
		}
	}

	fullPath := workspacePath + "/" + ec.Path
	_, err := os.Stat(fullPath)
	if err == nil {
		return EvalResult{
			Type:    "file_exists",
			Score:   1.0,
			Weight:  ec.Weight,
			Passed:  true,
			Details: fmt.Sprintf("file %q exists at %s", ec.Path, fullPath),
		}
	}

	return EvalResult{
		Type:    "file_exists",
		Score:   0,
		Weight:  ec.Weight,
		Passed:  false,
		Details: fmt.Sprintf("file %q not found at %s", ec.Path, fullPath),
	}
}

// evalCost computes token cost from Gateway metadata.
func evalCost(ec EvalConfig, resp GatewayResponse) EvalResult {
	if resp.Tokens.Available {
		// Real token data available
		cost := estimateCostFromTokens(resp.Model, resp.Tokens)
		return EvalResult{
			Type:    "cost",
			Score:   cost,
			Weight:  ec.Weight,
			Passed:  true,
			Details: fmt.Sprintf("$%.6f (%d tokens, model: %s)", cost, resp.Tokens.Total, resp.Model),
		}
	}

	// Fallback: estimate from response length
	estimatedTokens := len(resp.Text) / 4 // rough ~4 chars per token
	cost := float64(estimatedTokens) * 0.000003 // rough estimate at $3/M tokens
	return EvalResult{
		Type:    "cost",
		Score:   cost,
		Weight:  ec.Weight,
		Passed:  true,
		Details: fmt.Sprintf("~$%.6f (estimated from %d chars, no token data)", cost, len(resp.Text)),
	}
}

// estimateCostFromTokens computes cost using a simple price table.
func estimateCostFromTokens(model string, tokens TokenUsage) float64 {
	// Price per token (input, output) in USD
	// These are rough approximations; users can contribute better pricing
	inputPrice := 0.000003  // $3/M tokens default
	outputPrice := 0.000015 // $15/M tokens default

	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "gpt-4o"):
		inputPrice, outputPrice = 0.0000025, 0.00001
	case strings.Contains(m, "gpt-4"):
		inputPrice, outputPrice = 0.00003, 0.00006
	case strings.Contains(m, "claude-3-5-sonnet"), strings.Contains(m, "claude-sonnet"):
		inputPrice, outputPrice = 0.000003, 0.000015
	case strings.Contains(m, "claude-3-5-haiku"), strings.Contains(m, "claude-haiku"):
		inputPrice, outputPrice = 0.0000008, 0.000004
	case strings.Contains(m, "claude-opus"):
		inputPrice, outputPrice = 0.000015, 0.000075
	case strings.Contains(m, "gemini"):
		inputPrice, outputPrice = 0.0000005, 0.0000015
	case strings.Contains(m, "ollama"), strings.Contains(m, "local"):
		return 0 // local models are free
	}

	return float64(tokens.Input)*inputPrice + float64(tokens.Output)*outputPrice
}

// evalLatency records wall-clock time. Score IS the latency in seconds
// (not 0-1 normalized, since lower is better and there's no fixed scale).
func evalLatency(ec EvalConfig, wallClockSeconds float64) EvalResult {
	return EvalResult{
		Type:    "latency",
		Score:   wallClockSeconds,
		Weight:  ec.Weight,
		Passed:  true,
		Details: fmt.Sprintf("%.2fs wall clock", wallClockSeconds),
	}
}

// evalFormatBullets checks that the response has exactly 3 bullet points,
// each under 20 words. This is a built-in evaluator for the instruction_following task.
func evalFormatBullets(ec EvalConfig, text string) EvalResult {
	lines := strings.Split(strings.TrimSpace(text), "\n")

	// Find bullet lines: -, *, •, numbered (with any whitespace after)
	var bullets []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		// Match: - text, * text, • text, 1. text, 1) text (with any whitespace)
		isBullet := false
		for _, prefix := range []string{"-", "*", "•", "1.", "2.", "3.", "4.", "5.", "1)", "2)", "3)", "4)", "5)"} {
			if strings.HasPrefix(trimmed, prefix) {
				rest := strings.TrimLeft(trimmed[len(prefix):], " \t")
				if rest != "" {
					isBullet = true
					break
				}
			}
		}
		if isBullet {
			bullets = append(bullets, trimmed)
		}
	}

	score := 0.0
	details := []string{}

	// Check: exactly 3 bullets
	if len(bullets) == 3 {
		score += 0.5
		details = append(details, "3 bullets: yes")
	} else {
		details = append(details, fmt.Sprintf("3 bullets: no (%d found)", len(bullets)))
	}

	// Check: each bullet under 20 words (strip bullet prefix before counting)
	allUnder20 := true
	for i, b := range bullets {
		// Strip bullet prefix before counting words
		stripped := b
		for _, prefix := range []string{"-", "*", "•", "1.", "2.", "3.", "4.", "5.", "1)", "2)", "3)", "4)", "5)"} {
			if strings.HasPrefix(stripped, prefix) {
				stripped = strings.TrimLeft(stripped[len(prefix):], " \t")
				break
			}
		}
		words := len(strings.Fields(stripped))
		if words > 20 {
			allUnder20 = false
			details = append(details, fmt.Sprintf("bullet %d: %d words (over 20)", i+1, words))
		}
	}
	if allUnder20 && len(bullets) > 0 {
		score += 0.5
		details = append(details, "all bullets under 20 words: yes")
	}

	return EvalResult{
		Type:    "format_bullets",
		Score:   score,
		Weight:  ec.Weight,
		Passed:  score == 1.0,
		Details: strings.Join(details, "; "),
	}
}

// evalExecCheck runs a shell command in the workspace and scores based on exit code.
// This unlocks SWE-bench, Exercism, and any test-suite-based benchmark.
func evalExecCheck(ec EvalConfig, workspacePath string) EvalResult {
	if workspacePath == "" {
		return EvalResult{
			Type:    "exec_check",
			Score:   0,
			Weight:  ec.Weight,
			Passed:  false,
			Details: "no workspace path specified",
		}
	}

	// ec.Path contains the command to run (e.g., "python -m pytest two_fer_test.py")
	cmd := exec.Command("bash", "-c", ec.Path)
	cmd.Dir = workspacePath
	output, err := cmd.CombinedOutput()

	if err == nil {
		return EvalResult{
			Type:    "exec_check",
			Score:   1.0,
			Weight:  ec.Weight,
			Passed:  true,
			Details: fmt.Sprintf("command passed: %s", ec.Path),
		}
	}

	// Truncate output for details
	outStr := string(output)
	if len(outStr) > 500 {
		outStr = outStr[:500] + "..."
	}

	return EvalResult{
		Type:    "exec_check",
		Score:   0,
		Weight:  ec.Weight,
		Passed:  false,
		Details: fmt.Sprintf("command failed: %s\n%s", ec.Path, outStr),
	}
}

// evalRegexReject checks that NONE of the regex patterns match the response text.
// Inverse of exact_match — used for "must NOT contain" checks (e.g., em dashes, sycophantic openers).
// Score is 1.0 if no patterns match, 0.0 if any pattern matches. Partial scoring by ratio of non-matches.
func evalRegexReject(ec EvalConfig, text string) EvalResult {
	if len(ec.Patterns) == 0 {
		return EvalResult{Type: "regex_reject", Score: 1.0, Weight: ec.Weight, Passed: true, Details: "no reject patterns specified"}
	}

	rejected := 0
	var violations []string
	for _, pattern := range ec.Patterns {
		re, err := regexp.Compile(pattern)
		if err != nil {
			violations = append(violations, fmt.Sprintf("invalid regex %q: %v", pattern, err))
			continue
		}
		if re.MatchString(text) {
			rejected++
			violations = append(violations, fmt.Sprintf("matched reject pattern %q", pattern))
		}
	}

	passed := rejected == 0
	score := float64(len(ec.Patterns)-rejected) / float64(len(ec.Patterns))
	details := fmt.Sprintf("%d/%d reject patterns matched", rejected, len(ec.Patterns))
	if len(violations) > 0 {
		details += "; " + strings.Join(violations, "; ")
	}

	return EvalResult{
		Type:    "regex_reject",
		Score:   score,
		Weight:  ec.Weight,
		Passed:  passed,
		Details: details,
	}
}

// evalResponseCheck runs a shell command with the response text piped to stdin.
// Exit 0 = pass, non-zero = fail. Unlike exec_check (which only gets workspace path),
// this evaluator passes the actual response text for content validation.
// ec.Path contains the command to run (e.g., a validation script path).
func evalResponseCheck(ec EvalConfig, text string, workspacePath string) EvalResult {
	if ec.Path == "" {
		return EvalResult{
			Type:    "response_check",
			Score:   0,
			Weight:  ec.Weight,
			Passed:  false,
			Details: "no command specified in Path",
		}
	}

	cmd := exec.Command("bash", "-c", ec.Path)
	if workspacePath != "" {
		cmd.Dir = workspacePath
	}
	cmd.Stdin = strings.NewReader(text)
	output, err := cmd.CombinedOutput()

	if err == nil {
		return EvalResult{
			Type:    "response_check",
			Score:   1.0,
			Weight:  ec.Weight,
			Passed:  true,
			Details: fmt.Sprintf("response check passed: %s", ec.Path),
		}
	}

	outStr := string(output)
	if len(outStr) > 500 {
		outStr = outStr[:500] + "..."
	}

	return EvalResult{
		Type:    "response_check",
		Score:   0,
		Weight:  ec.Weight,
		Passed:  false,
		Details: fmt.Sprintf("response check failed: %s\n%s", ec.Path, outStr),
	}
}

// ComputeCorrectness aggregates correctness-related evaluator scores.
func ComputeCorrectness(results []EvalResult) float64 {
	return weightedAverage(results, "exact_match", "format_bullets", "exec_check", "gaia_exact", "regex_reject", "response_check")
}

// ComputeToolAccuracy aggregates tool-related evaluator scores.
func ComputeToolAccuracy(results []EvalResult) float64 {
	return weightedAverage(results, "tool_invoked", "file_exists")
}

func weightedAverage(results []EvalResult, types ...string) float64 {
	typeSet := make(map[string]bool)
	for _, t := range types {
		typeSet[t] = true
	}

	sumWeighted := 0.0
	sumWeight := 0.0
	for _, r := range results {
		if typeSet[r.Type] {
			sumWeighted += r.Score * r.Weight
			sumWeight += r.Weight
		}
	}
	if sumWeight == 0 {
		return 0
	}
	return sumWeighted / sumWeight
}
