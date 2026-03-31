# Adding Benchmark Tasks

Tasks are currently embedded as Go structs in `tasks.go`. To add a new task:

## 1. Define the task

Add a new `Task` struct to the slice returned by `BuiltinTasks()`:

```go
{
    ID:       "my_new_task",
    Name:     "My New Task Description",
    Category: "tool_use",  // or "correctness"
    Tags:     []string{"tag1", "tag2"},
    Prompt:   `The prompt that gets sent to the agent.`,
    TimeBudget: 90 * time.Second,
    Evaluators: []EvalConfig{
        {Type: "exact_match", Patterns: []string{`expected_pattern`}, Weight: 1.0},
        {Type: "tool_invoked", ToolName: "expected_tool", Weight: 1.0},
        {Type: "cost", Weight: 0.3},
        {Type: "latency", Weight: 0.3},
    },
},
```

## 2. Choose the right evaluators

Pick evaluators that test config-dependent behavior, not generic LLM capability:

| If you want to test... | Use evaluator | Why |
|------------------------|---------------|-----|
| Skill discovery | `tool_invoked` | Does the workspace have the right skill? |
| Tool chaining | Multiple `tool_invoked` | Does the agent chain tools correctly? |
| Output generation | `file_exists` | Did the agent produce a file? |
| Instruction following | `format_bullets` or `exact_match` | Does SOUL.md affect format adherence? |
| Factual correctness | `exact_match` with regex | Does the response contain the right answer? |

Always include `cost` and `latency` for comparison across setups.

## 3. Rebuild

```bash
go build -o clawbench .
```

## 4. Task design principles

- **Test the config, not the model.** A task that scores identically across all configs using the same model is useless. The task should exercise skills, tool routing, SOUL.md instructions, or AGENTS.md rules.
- **Fixed time budgets.** Set `TimeBudget` to something reasonable for the task. This normalizes comparison.
- **Deterministic evaluation where possible.** Regex patterns and file checks are deterministic. Use `--repeat N` for tasks where LLM nondeterminism matters.

## Future: YAML task loading

When the community needs user-contributed tasks without recompilation, YAML loading will be added. The task struct maps directly to YAML:

```yaml
id: my_new_task
name: "My New Task"
category: tool_use
tags: [tag1, tag2]
prompt: |
  The prompt text.
time_budget_seconds: 90
evaluators:
  - type: exact_match
    patterns: ["expected_pattern"]
    weight: 1.0
```
