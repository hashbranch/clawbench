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

## 6. Adapting public benchmarks (GAIA example)

ClawBench supports embedding tasks from published academic benchmarks. The GAIA Level 1 tasks (`gaia_l1_*`) are a good example of this pattern:

1. **Source questions** from the benchmark dataset (e.g., HuggingFace, paper appendices)
2. **Filter** for questions answerable with your agent's tools (skip image/audio/PDF tasks if your setup doesn't support them)
3. **Use the right evaluator**: `gaia_exact` for GAIA-style exact string matching, `exact_match` for regex-based checks
4. **Add category and tags** matching the benchmark (`gaia_l1`, tags: `gaia`, `level1`, etc.)
5. **Set appropriate time budgets** — multi-step reasoning tasks may need 120s+

The `gaia_exact` evaluator implements normalized comparison (lowercase, strip whitespace/punctuation, numeric normalization) matching the official GAIA scorer. Use it for any task where the ground truth is a short, unambiguous factual answer.

See [docs/gaia.md](gaia.md) for details on the GAIA benchmark and how to add more questions from the full dataset.


