# GAIA Benchmark Integration

## What is GAIA?

[GAIA (General AI Assistants)](https://arxiv.org/abs/2311.12983) is a benchmark by Mialon et al. (2023) that evaluates AI assistants on real-world tasks requiring reasoning, web browsing, multi-modality handling, and tool-use proficiency.

Key characteristics:
- **466 questions** with unambiguous factual answers
- **3 difficulty levels**: Level 1 (simple, ≤5 steps), Level 2 (5-10 steps, multi-tool), Level 3 (arbitrary complexity)
- **Human baseline: 92%** vs GPT-4 with plugins: ~15% (at launch)
- Questions are "conceptually simple for humans yet challenging for AI"

The original dataset is available at [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA) (gated, requires access request).

## ClawBench GAIA Tasks

ClawBench includes **15 GAIA-style Level 1 tasks** (`gaia_l1_001` through `gaia_l1_015`) that follow GAIA's philosophy:

- Questions have a single, unambiguous answer
- Answers are short (a number, a word, or a brief phrase)
- Questions test reasoning, calculation, factual lookup, and multi-step logic
- No attached files required (text-only questions)

### Question Categories

| Tag | Description | Example |
|-----|-------------|---------|
| `math` | Arithmetic, conversion, calculation | Sum of primes, unit conversion |
| `factual` | Knowledge lookup | Chemical elements, historical dates |
| `web_search` | May require web search | Wikipedia facts, geographic knowledge |
| `reasoning` | Logic and deduction | Date calculation, Roman numerals |
| `coding` | String/number manipulation | Vowel counting, base conversion |
| `multi_step` | Multiple operations needed | Tax calculation |
| `conversion` | Unit or base conversion | Cups to liters, hex to decimal |

### Scoring: `gaia_exact` Evaluator

ClawBench implements a `gaia_exact` evaluator that mirrors the official GAIA scoring function:

1. **String answers**: Lowercased, whitespace removed, punctuation stripped, then compared
2. **Numeric answers**: Common formatting ($, %, commas) removed, compared as floats
3. **List answers** (comma/semicolon-separated): Element-by-element comparison using the above rules

The evaluator also attempts to extract a concise answer from verbose agent responses by looking for patterns like "FINAL ANSWER:", "The answer is:", or falling back to the last non-empty line.

## Published Baselines (HAL Leaderboard)

The [HAL GAIA Leaderboard](https://hal.cs.princeton.edu/gaia) evaluates on the full 165-question public validation set. Top results as of early 2026:

| Agent | Model | Overall | Level 1 | Level 2 | Level 3 |
|-------|-------|---------|---------|---------|---------|
| HAL Generalist | Claude Sonnet 4.5 | 74.55% | 82.07% | 72.68% | 65.39% |
| HAL Generalist | Claude Opus 4.1 High | 68.48% | 71.70% | 70.93% | 53.85% |
| HAL Generalist | o4-mini Low | 58.18% | 71.70% | 51.16% | 53.85% |
| HF Open Deep Research | GPT-5 Medium | 62.80% | 73.58% | 62.79% | 38.46% |

**Note:** ClawBench's GAIA tasks are a curated subset of GAIA-style questions, not the full validation set. Scores are not directly comparable to leaderboard numbers but serve as a directional signal for how well your OpenClaw setup handles this class of problem.

## Running GAIA Tasks

```bash
# Run all GAIA Level 1 tasks
clawbench run --label "my-setup" --task gaia_l1_001

# Run all tasks (includes GAIA)
clawbench run --label "full-bench"

# List GAIA tasks
clawbench list | grep gaia
```

## Adapting More GAIA Questions

If you have access to the full GAIA dataset on HuggingFace:

1. Request access at [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA)
2. Filter for Level 1 validation questions with no attached files (`file_name` is empty)
3. Add them to `GAIATasks()` in `tasks.go` following the existing pattern
4. Use the `gaia_exact` evaluator with the `Final answer` field as the ground truth

See [adding-tasks.md](adding-tasks.md) for the general process of adding benchmark tasks.

## References

- Paper: [GAIA: a benchmark for General AI Assistants](https://arxiv.org/abs/2311.12983) (Mialon et al., 2023)
- Dataset: [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA)
- Leaderboard: [huggingface.co/spaces/gaia-benchmark/leaderboard](https://huggingface.co/spaces/gaia-benchmark/leaderboard)
- HAL Leaderboard: [hal.cs.princeton.edu/gaia](https://hal.cs.princeton.edu/gaia)
