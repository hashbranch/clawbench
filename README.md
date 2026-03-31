# ClawBench

Objective benchmarking for OpenClaw configurations. Stop arguing about which setup is better. Get receipts.

ClawBench connects to your running OpenClaw Gateway, runs standardized benchmark tasks, and scores the results. Share result files with your team and compare setups side-by-side.

## Install

```bash
go install github.com/tommerkle/clawbench@latest
```

Or build from source:

```bash
git clone https://github.com/tommerkle/clawbench.git
cd clawbench
go build -o clawbench .
```

## Usage

```bash
# List available benchmark tasks
clawbench list

# Run all benchmarks against your local Gateway
clawbench run --label "my-setup-v2"

# Run a specific task with 3 repeats for statistical rigor
clawbench run --task skill_tool_chain --repeat 3

# Compare two result files
clawbench compare results/setup-a.json results/setup-b.json
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--gateway` | `ws://127.0.0.1:18789` | Gateway WebSocket URL |
| `--token` | `$OPENCLAW_AUTH_TOKEN` | Auth token |
| `--label` | timestamp | Label for this run |
| `--task` | all | Run specific task only |
| `--repeat` | 1 | Repeat each task N times |
| `--workspace` | | Path to OpenClaw workspace (for file checks) |
| `--output` | `results/<label>.json` | Output file path |

## Benchmark Tasks

**Skill Discovery + Tool Chaining** — Tests whether your setup can find and chain the right skills (weather lookup + file write). Exercises skill discovery, tool chaining, and output generation.

**Instruction Following** — Tests whether your SOUL.md and AGENTS.md configuration affects how well the agent follows structured instructions (exact bullet count, word limits).

## Metrics

Each task is scored on independent metrics (not composited into a single score):

- **Correctness** — Did the agent produce the right answer?
- **Tool Accuracy** — Did it use the right tools/skills?
- **Latency** — How fast from prompt to response?
- **Cost** — Token cost (real if available, estimated otherwise)

## How Comparison Works

1. You run `clawbench run` on your machine
2. Your teammate runs it on theirs
3. Share the JSON result files
4. `clawbench compare` shows a side-by-side table with deltas

Results capture config metadata (model, temperature, Gateway version) so you know what you're comparing.

## License

MIT
