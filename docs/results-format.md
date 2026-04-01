# Results JSON Format

Each `clawbench run` produces a JSON file with this structure:

```json
{
  "version": "0.5.0",
  "timestamp": "2026-03-31T14:30:00Z",
  "label": "my-setup-v2",
  "config": {
    "label": "my-setup-v2",
    "model": "anthropic/claude-sonnet-4-6",
    "temperature": 0.7,
    "gateway_url": "ws://127.0.0.1:18789",
    "gateway_version": "2026.3.28",
    "workspace_hash": ""
  },
  "results": [
    {
      "run_id": "20260331_143000_a1b2c3d4",
      "task_id": "skill_tool_chain",
      "timestamp": "2026-03-31T14:30:05Z",
      "correctness": 0.75,
      "tool_accuracy": 1.0,
      "wall_clock_seconds": 4.2,
      "total_tokens": 620,
      "cost_usd": 0.003,
      "num_tool_calls": 2,
      "tools_used": ["exec", "write"],
      "config": {
        "model": "anthropic/claude-sonnet-4-6",
        "temperature": 0.7
      },
      "eval_results": [
        {
          "type": "tool_invoked",
          "score": 1.0,
          "weight": 1.0,
          "passed": true,
          "details": "tool \"exec\" invoked (from trace)"
        },
        {
          "type": "tool_invoked",
          "score": 1.0,
          "weight": 1.0,
          "passed": true,
          "details": "tool \"write\" invoked (from trace)"
        },
        {
          "type": "file_exists",
          "score": 1.0,
          "weight": 1.0,
          "passed": true,
          "details": "file \"weather_haiku.txt\" exists at /path/to/workspace/weather_haiku.txt"
        }
      ],
      "raw_response": "I looked up the weather in San Francisco...",
      "is_error": false
    }
  ],
  "summary": {
    "total_tasks": 2,
    "total_runs": 2,
    "repeat_count": 1,
    "avg_correctness": 0.85,
    "avg_latency_seconds": 3.5,
    "total_cost_usd": 0.006
  }
}
```

## Config Metadata

The `config` object captures what was being benchmarked. Without this, comparing results from different machines is meaningless.

| Field | Source | Description |
|-------|--------|-------------|
| `label` | `--label` flag | User-provided name for this run |
| `model` | Gateway response | Model that served the responses |
| `temperature` | Gateway response | Sampling temperature (0 if not reported) |
| `gateway_url` | `--gateway` flag | Gateway WebSocket URL |
| `gateway_version` | Connect handshake | OpenClaw version (e.g. `2026.3.28`) |
| `workspace_hash` | (future) | Hash of SOUL.md + AGENTS.md + skills |

## Repeated Runs

With `--repeat N`, each task runs N times. The `summary` uses median values for aggregation. Individual run results are all included in the `results` array.

## Comparison

`clawbench compare a.json b.json` reads two result files and computes per-task deltas using median values when multiple runs exist per task.
