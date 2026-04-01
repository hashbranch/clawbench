# Evaluators

ClawBench has 6 built-in evaluators. Each produces a score (0.0-1.0) and a human-readable details string.

## exact_match

Checks if the response text matches regex patterns.

- **Config**: `Patterns []string` — list of regex patterns
- **Scoring**: `matched / total patterns`
- **Edge cases**: Invalid regex scores 0 with a warning. Empty response scores 0.

```
Input:  "The weather in SF is 65°F and sunny"
Pattern: "(?i)sunny|clear"
Score:  1.0
```

## tool_invoked

Checks if a specific tool/skill was used during the response.

- **Config**: `ToolName string` — expected tool name
- **Primary**: Checks structured tool call data from Gateway agent events. These arrive as `event: "agent"` frames with `stream: "tool"` and the tool name in `data.name`. Score: 1.0.
- **Fallback**: If no structured tool data is available, checks response text for heuristic mentions like "used exec" or "called write". Score: 0.5 (lower confidence).
- Case-insensitive matching.

## file_exists

Checks if the agent created an expected output file.

- **Config**: `Path string` — relative file path
- **Requires**: `--workspace` flag pointing to the OpenClaw workspace directory
- Without `--workspace`, scores 0 with "no workspace path specified"

## cost

Computes token cost from Gateway metadata.

- **With token data**: Uses a price table keyed by model name (GPT-4o, Claude Sonnet, Claude Opus, Haiku, Gemini, Ollama, etc.)
- **Without token data**: Estimates from response character length at ~4 chars/token and $3/M tokens. The Gateway doesn't always include token counts in its responses, so this fallback is common.
- **Local models** (Ollama): Cost is $0

The score field contains the raw USD cost (not normalized 0-1). This is intentional since cost is reported as a dollar amount, not a pass/fail.

## latency

Wall-clock time from prompt send to response complete.

- Score is the raw seconds value
- Always passes (latency is informational, not pass/fail)

## format_bullets

Built-in evaluator for the instruction_following task. Checks structured output format.

- Checks: exactly 3 bullet points (0.5 if correct count)
- Checks: each bullet under 20 words, excluding the bullet prefix marker (0.5 if all pass)
- Recognizes bullet markers: `-`, `*`, `•` (Unicode bullet), `1.`–`5.`, `1)`–`5)` (numbered lists above 5 are not currently matched)
- Empty lines between bullets are ignored (handles double-newline separated formats)

## Composite Scores

Individual evaluator results are aggregated into two composite dimensions:

- **Correctness** = weighted average of `exact_match` + `format_bullets` evaluators
- **Tool Accuracy** = weighted average of `tool_invoked` + `file_exists` evaluators

Weights are relative within each dimension. An evaluator with weight 1.0 counts 2x one with weight 0.5.

## Adding New Evaluators

Add a new case to the switch in `evaluator.go:runEvaluator()`, implement the scoring function, and add it to the appropriate composite dimension in `ComputeCorrectness()` or `ComputeToolAccuracy()`. Write tests in `evaluator_test.go`.
