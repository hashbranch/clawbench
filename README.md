# ClawBench

Objective benchmarking for OpenClaw configurations. Stop arguing about which setup is better. Get receipts.

ClawBench connects to your running OpenClaw Gateway, runs standardized benchmark tasks, and scores the results. Share result files with your team and compare setups side-by-side.

## Install

```bash
go install github.com/hashbranch/clawbench@latest
```

Or build from source:

```bash
git clone https://github.com/hashbranch/clawbench.git
cd clawbench
go build -o clawbench .
```

## Quick Start

ClawBench needs a running OpenClaw Gateway. It creates isolated sessions per task, so it won't interfere with your active agent conversations.

**Using the CLI backend (default, recommended):**

```bash
# Requires openclaw CLI on PATH and Gateway running
clawbench run --label "my-setup"
```

**Using the WebSocket backend (direct connection):**

```bash
clawbench run --mode websocket --token "$OPENCLAW_AUTH_TOKEN" --label "my-setup"
```

### Device Identity

ClawBench auto-detects your device identity from `~/.openclaw/identity/device.json` for scope binding when using WebSocket mode. This is the same Ed25519 key pair used by the OpenClaw CLI. Without it, the Gateway may reject scope requests or limit access.

The CLI backend bypasses this since the `openclaw` CLI handles its own auth.

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
| `--mode` | `cli` | Backend mode: `cli` (openclaw CLI) or `websocket` (direct WS) |
| `--gateway` | `ws://127.0.0.1:18789` | Gateway WebSocket URL (websocket mode only) |
| `--token` | `$OPENCLAW_AUTH_TOKEN` | Auth token (websocket mode only) |
| `--label` | timestamp | Label for this run |
| `--hf-token` | `$HF_TOKEN` | HuggingFace token for official GAIA questions |
| `--gaia-only` | false | Run only official GAIA tasks (requires `--hf-token`) |
| `--task` | all | Run specific task only |
| `--repeat` | 1 | Repeat each task N times |
| `--workspace` | | Path to OpenClaw workspace (for file checks) |
| `--output` | `results/<label>.json` | Output file path |

## Benchmark Tasks

**Skill Discovery + Tool Chaining** (`skill_tool_chain`) — Prompts the agent to look up weather and write a haiku to a file. Exercises tool discovery (`exec` for weather fetching), tool chaining (`write` for file creation), and output generation (`file_exists` check). 90s budget.

**Instruction Following** (`instruction_following`) — Tests structured output adherence: exactly 3 bullet points, each under 20 words. Exercises whether SOUL.md and AGENTS.md configuration affects format compliance. 60s budget.

### ClawBench Original Tasks

15 reasoning and knowledge tasks (`cb_reasoning_001` through `cb_reasoning_015`) authored by ClawBench. These follow the GAIA benchmark style (short factual answers, exact match) but are NOT from the official dataset. They serve as a baseline that runs without any external dependencies.

### Official GAIA Level 1 Tasks (Runtime Fetch)

When you provide a HuggingFace token, ClawBench fetches real [GAIA benchmark](https://arxiv.org/abs/2311.12983) Level 1 validation questions at runtime. The GAIA dataset is gated — questions can't be redistributed, so they're downloaded fresh each run.

```bash
# Include official GAIA alongside built-in tasks
clawbench run --hf-token "$HF_TOKEN" --label "full-bench"

# Run ONLY official GAIA tasks
clawbench run --hf-token "$HF_TOKEN" --gaia-only --label "gaia-bench"
```

Questions requiring file attachments or multimedia (images, video, audio) are automatically filtered out — only text-based questions are used.

Each task uses the `gaia_exact` evaluator, which implements GAIA's official scoring: normalized exact string matching with whitespace/punctuation removal, numeric comparison, and list comparison. 120s budget per task.

**Published GAIA baselines** (full 165-question validation set): Claude Sonnet 4.5 achieves 74.55% overall (82% on Level 1). See [docs/gaia.md](docs/gaia.md) for details and leaderboard comparison.

## Metrics

Each task is scored on independent metrics (not composited into a single score):

- **Correctness** — Did the agent produce the right answer?
- **Tool Accuracy** — Did it use the right tools/skills?
- **Latency** — How fast from prompt to response?
- **Cost** — Token cost (real if available, estimated from response char length otherwise)

## How Comparison Works

1. You run `clawbench run` on your machine
2. Your teammate runs it on theirs
3. Share the JSON result files
4. `clawbench compare` shows a side-by-side table with deltas

Results capture config metadata (model, Gateway version, gateway URL) so you know what you're comparing.

## Session Isolation

Each benchmark task runs in its own isolated session via `sessions.create`. This means:

- Benchmark runs don't pollute your active agent conversations
- Each task gets a clean context (no bleed between tasks)
- Sessions are labeled `ClawBench - <seq> - <timestamp>` for easy identification

## License

MIT
