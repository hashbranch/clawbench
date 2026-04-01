# ClawBench

Objective benchmarking for OpenClaw configurations. Stop arguing about which setup is better. Get receipts.

ClawBench connects to your running OpenClaw Gateway, runs standardized benchmark tasks, and scores the results. Share result files with your team and compare setups side-by-side.

## Install

Download the latest binary from [GitHub Releases](https://github.com/hashbranch/clawbench/releases):

```bash
curl -sL https://github.com/hashbranch/clawbench/releases/latest/download/clawbench-darwin-arm64 -o clawbench
chmod +x clawbench
```

Or build from source:

```bash
git clone https://github.com/hashbranch/clawbench.git
cd clawbench
go build -o clawbench .
```

## Usage

```bash
# Run builtin benchmark tasks
clawbench run --mode websocket --token "your-token" --label "my-setup"

# Run PinchBench-adapted tasks (real-world agent scenarios)
clawbench run --benchmark pinchbench --mode websocket --token "your-token" --label "my-setup"

# Run Exercism coding benchmark (34 Python exercises)
clawbench run --benchmark exercism --mode websocket --token "your-token" --label "my-setup"

# Run a specific task
clawbench run --benchmark pinchbench --task pinch/weather_script

# Compare two result files
clawbench compare results/setup-a.json results/setup-b.json

# List available tasks
clawbench list
```

### Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--mode` | `cli` | Backend: `websocket` (recommended) or `cli` |
| `--benchmark` | builtin | Suite: `builtin`, `pinchbench`, or `exercism` |
| `--gateway` | `ws://127.0.0.1:18789` | Gateway WebSocket URL |
| `--token` | `$OPENCLAW_AUTH_TOKEN` | Auth token |
| `--label` | timestamp | Label for this run |
| `--task` | all | Run specific task only |
| `--repeat` | 1 | Repeat each task N times |
| `--workspace` | | OpenClaw workspace path (for file checks) |
| `--output` | `results/<label>.json` | Output file path |
| `--debug` | false | Dump raw WebSocket frames |

## Benchmark Suites

### Builtin (2 tasks)

Quick smoke test for your setup.

- **Skill Discovery + Tool Chaining** -- Can your agent find and chain tools (weather + file write)?
- **Instruction Following** -- Does your SOUL.md affect how well the agent follows structured instructions?

### PinchBench (8 tasks)

Real-world agent tasks adapted from [PinchBench](https://github.com/pinchbench/skill). Covers coding, file operations, comprehension, multi-step workflows, and organization.

- **Sanity Check** -- fail-fast gate, aborts if the agent can't respond
- **Weather Script** -- create a working Python script using an API
- **File Structure** -- create a project scaffold (src/, README, .gitignore)
- **Search and Replace** -- find and replace across config files
- **Multi-step Workflow** -- read config, write code, write docs
- **Memory Retrieval** -- read a file, extract a fact, save the answer
- **Email Triage** -- categorize and prioritize 10 emails
- **Blog Post** -- write a structured 500-word blog post

PinchBench originally benchmarks models (holding config constant). ClawBench adapts these tasks to benchmark configurations (holding model constant).

### GAIA-Style Reasoning (15 tasks)

15 reasoning and knowledge tasks following the [GAIA benchmark](https://arxiv.org/abs/2311.12983) philosophy: short unambiguous answers, real-world knowledge, multi-step reasoning. Scored with the `gaia_exact` evaluator implementing official GAIA scoring (normalized exact string matching, numeric comparison, list comparison). Included in the default task set.

Optional: provide a HuggingFace token to also fetch real GAIA Level 1 questions from the gated dataset. See [docs/gaia.md](docs/gaia.md) for published baselines (Claude Sonnet 4.5 achieves 82% on Level 1).

### Exercism (34 tasks)

Python coding exercises from the [Aider Polyglot Benchmark](https://github.com/Aider-AI/polyglot-benchmark). Auto-downloaded on first run. Each exercise has a stub file and a unittest test suite. Scores are comparable to published numbers on the [Aider Leaderboard](https://aider.chat/docs/leaderboards/).

## Metrics

Each task is scored on independent metrics (not composited into a single score):

- **Correctness** -- Did the agent produce the right answer?
- **Tool Accuracy** -- Did it use the right tools/skills?
- **Latency** -- Wall-clock time from prompt to response
- **Cost** -- Token cost (real from Gateway, or estimated)
- **Efficiency** -- Score per 1K tokens, score per dollar

## How Comparison Works

1. You run `clawbench run` on your machine
2. Your teammate runs it on theirs
3. Share the JSON result files
4. `clawbench compare` shows a side-by-side table with deltas

Results capture config metadata (model, temperature, Gateway version) so you know what you're comparing.

## Acknowledgments

- [PinchBench](https://github.com/pinchbench/skill) -- Task designs and real-world agent scenarios adapted for configuration benchmarking. Original project benchmarks LLM models in OpenClaw.
- [Aider Polyglot Benchmark](https://github.com/Aider-AI/polyglot-benchmark) -- Exercism Python exercises used for the coding benchmark suite.
- [Karpathy's AutoResearch](https://github.com/karpathy/autoresearch) -- Inspiration for fixed time budgets and immutable evaluation harness design.
- [Meta-Harness](https://arxiv.org/html/2603.28052v1) -- Inspiration for full execution trace capture over scalar scores.
- [OpenClaw](https://github.com/openclaw/openclaw) -- The agent platform this tool benchmarks.

## License

MIT
