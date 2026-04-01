# GAIA Benchmark Integration

## What is GAIA?

[GAIA (General AI Assistants)](https://arxiv.org/abs/2311.12983) is a benchmark by Mialon et al. (2023) that evaluates AI assistants on real-world tasks requiring reasoning, web browsing, multi-modality handling, and tool-use proficiency.

Key characteristics:
- **466 questions** with unambiguous factual answers
- **3 difficulty levels**: Level 1 (simple, ≤5 steps), Level 2 (5-10 steps, multi-tool), Level 3 (arbitrary complexity)
- **Human baseline: 92%** vs GPT-4 with plugins: ~15% (at launch)
- Questions are "conceptually simple for humans yet challenging for AI"

The original dataset is available at [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA) (gated, requires access request).

## Two Task Sources

### ClawBench Originals (always available)

15 reasoning/knowledge tasks (`cb_reasoning_001` through `cb_reasoning_015`) that follow the GAIA style — short factual answers, exact-match scoring. These are authored by ClawBench and don't require any external access. They serve as a baseline for how your OpenClaw setup handles this class of problem.

### Official GAIA Questions (runtime fetch)

Real GAIA Level 1 validation questions fetched from HuggingFace at runtime. The dataset is **gated** — ClawBench cannot redistribute the questions, so they must be downloaded each time you run.

**Why runtime fetch?** The GAIA dataset license prohibits redistribution. By fetching at runtime with the user's own token, we comply with the license while still enabling standardized benchmarking against the official questions.

## Getting Access

1. Create a [HuggingFace account](https://huggingface.co/join)
2. Request access at [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA) (approval is typically fast)
3. Create an access token at [huggingface.co/settings/tokens](https://huggingface.co/settings/tokens) (read access is sufficient)

## Running GAIA Tasks

```bash
# Set token via env var (recommended)
export HF_TOKEN="hf_xxxxxxxxxxxxxxxxxxxx"

# Run all tasks including official GAIA
clawbench run --label "full-bench"

# Or pass token as flag
clawbench run --hf-token "$HF_TOKEN" --label "full-bench"

# Run ONLY official GAIA tasks (no built-in tasks)
clawbench run --hf-token "$HF_TOKEN" --gaia-only --label "gaia-only"

# Run a specific GAIA task
clawbench run --hf-token "$HF_TOKEN" --task gaia_l1_real_005

# Without a token, only built-in tasks run (ClawBench originals)
clawbench run --label "builtin-only"
```

## Question Filtering

Not all GAIA questions are suitable for text-based agent benchmarking. ClawBench automatically filters out questions that:

- **Require file attachments** (`file_name` field is non-empty)
- **Reference multimedia** (YouTube, video, audio, image, photo, screenshot, recording)
- **Reference specific file formats** (PDF, spreadsheet, Excel, .xlsx, .csv, .zip)
- **Have empty answers** (data quality issue)

This typically yields ~35-45 usable questions from the Level 1 validation split (out of ~53 total).

## Scoring: `gaia_exact` Evaluator

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

**Note:** ClawBench's official GAIA tasks use the Level 1 validation split filtered to text-only questions. Scores are not directly comparable to full leaderboard numbers but serve as a strong directional signal.

## References

- Paper: [GAIA: a benchmark for General AI Assistants](https://arxiv.org/abs/2311.12983) (Mialon et al., 2023)
- Dataset: [huggingface.co/datasets/gaia-benchmark/GAIA](https://huggingface.co/datasets/gaia-benchmark/GAIA)
- Leaderboard: [huggingface.co/spaces/gaia-benchmark/leaderboard](https://huggingface.co/spaces/gaia-benchmark/leaderboard)
- HAL Leaderboard: [hal.cs.princeton.edu/gaia](https://hal.cs.princeton.edu/gaia)
