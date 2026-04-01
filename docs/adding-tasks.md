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
        {Type: "tool_invoked", ToolName: "exec", Weight: 1.0},
        {Type: "tool_invoked", ToolName: "write", Weight: 1.0},
        {Type: "file_exists", Path: "output.txt", Weight: 1.0},
        {Type: "cost", Weight: 0.3},
        {Type: "latency", Weight: 0.3},
    },
},
```

## 2. Choose the right evaluators

Pick evaluators that test config-dependent behavior, not generic LLM capability:

| If you want to test... | Use evaluator | Example |
|------------------------|---------------|---------|
| Tool discovery | `tool_invoked` | `ToolName: "exec"` — did it shell out? |
| Tool chaining | Multiple `tool_invoked` | `exec` + `write` — did it chain correctly? |
| Output generation | `file_exists` | `Path: "weather_haiku.txt"` |
| Instruction following | `format_bullets` or `exact_match` | Does SOUL.md affect format adherence? |
| Factual correctness | `exact_match` with regex | Does the response contain the right answer? |

Always include `cost` and `latency` for comparison across setups.

Common tool names in OpenClaw: `exec`, `write`, `read`, `Edit`, `web_search`, `web_fetch`, `browser`, `image`, `tts`.

## 3. Session isolation

Each task runs in its own isolated session (via `sessions.create`). This means:

- Tasks don't bleed context into each other
- Benchmark runs don't affect active agent conversations
- Each task gets a clean slate

You don't need to manage sessions yourself — the runner handles this.

## 4. Rebuild

```bash
go build -o clawbench .
```

## 5. Task design principles

- **Test the config, not the model.** A task that scores identically across all configs using the same model is useless. The task should exercise skills, tool routing, SOUL.md instructions, or AGENTS.md rules.
- **Fixed time budgets.** Set `TimeBudget` to something reasonable for the task. This normalizes comparison.
- **Deterministic evaluation where possible.** Regex patterns and file checks are deterministic. Use `--repeat N` for tasks where LLM nondeterminism matters.


