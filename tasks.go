package main

import "time"

// BuiltinTasks returns the embedded benchmark task definitions.
// These test config-dependent behavior, not generic LLM capabilities.
func BuiltinTasks() []Task {
	return []Task{
		{
			ID:       "skill_tool_chain",
			Name:     "Skill Discovery + Tool Chaining",
			Category: "tool_use",
			Tags:     []string{"skill", "tool_chain", "workspace"},
			Prompt: `Find out what the weather is in San Francisco right now, then write a short haiku about it and save it to a file called weather_haiku.txt in my workspace.`,
			TimeBudget: 90 * time.Second,
			Evaluators: []EvalConfig{
				// Real agents use exec (curl/fetch) for weather, not a "weather" tool
				{Type: "tool_invoked", ToolName: "exec", Weight: 1.0},
				{Type: "tool_invoked", ToolName: "write", Weight: 1.0},
				{Type: "file_exists", Path: "weather_haiku.txt", Weight: 1.0},
				// Check response mentions weather/haiku/file creation
				{Type: "exact_match", Patterns: []string{`(?i)(weather|temperature|forecast|haiku|weather_haiku)`}, Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "instruction_following",
			Name:     "SOUL.md / AGENTS.md Instruction Adherence",
			Category: "correctness",
			Tags:     []string{"instructions", "workspace", "soul", "agents"},
			Prompt: `Summarize the key benefits of renewable energy in exactly 3 bullet points. Keep each bullet under 20 words.`,
			TimeBudget: 60 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "exact_match", Patterns: []string{`(?i)renewable|solar|wind|energy`}, Weight: 0.5},
				{Type: "format_bullets", Weight: 1.0}, // built-in: checks 3 bullets, each <20 words
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
	}
}

// ClawBenchOriginalTasks returns reasoning/knowledge tasks authored by ClawBench.
// These are inspired by the GAIA benchmark style (short factual answers, exact match)
// but are NOT from the official GAIA dataset. They serve as a baseline that doesn't
// require HuggingFace access.
func ClawBenchOriginalTasks() []Task {
	return []Task{
		{
			ID:       "cb_reasoning_001",
			Name:     "ClawBench: Simple Arithmetic",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math"},
			Prompt:   `What is the sum of the prime numbers between 20 and 40? Provide just the number as your final answer.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"120"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_002",
			Name:     "ClawBench: Unit Conversion",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math", "conversion"},
			Prompt:   `A recipe calls for 2.5 cups of flour. If 1 cup equals 236.588 milliliters, how many liters of flour is that? Round to 2 decimal places. Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"0.59"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_003",
			Name:     "ClawBench: Wikipedia Factual Lookup",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `What year was the Python programming language first released? Provide just the year.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"1991"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_004",
			Name:     "ClawBench: Chemical Element Lookup",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `What is the chemical symbol for the element with atomic number 74? Provide just the symbol.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"W"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_005",
			Name:     "ClawBench: Multi-Step Calculation",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math", "multi_step"},
			Prompt:   `If you buy 3 items costing $12.50, $8.75, and $15.00, and there is a 7% sales tax applied to the total, what is the total amount you pay? Give your answer in dollars, rounded to 2 decimal places. Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"38.79"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_006",
			Name:     "ClawBench: Historical Fact",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `Who was the first person to walk on the Moon, and in what year did it happen? Answer in the format: Firstname Lastname, YYYY`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Neil Armstrong, 1969"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_007",
			Name:     "ClawBench: Reverse Lookup",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `In the NATO phonetic alphabet, what word represents the letter 'Q'? Provide just the word.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Quebec"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_008",
			Name:     "ClawBench: Geographic Knowledge",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `What is the capital city of Australia? Provide just the city name.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Canberra"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_009",
			Name:     "ClawBench: String Manipulation",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "coding"},
			Prompt:   `How many vowels (a, e, i, o, u) are in the word "onomatopoeia"? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"8"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_010",
			Name:     "ClawBench: Date Calculation",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math"},
			Prompt:   `If January 1, 2024 was a Monday, what day of the week was March 1, 2024? Provide just the day name (e.g., Monday).`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Friday"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_011",
			Name:     "ClawBench: Roman Numerals",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math"},
			Prompt:   `What is the value of the Roman numeral MCMXCIV in Arabic numerals? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"1994"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_012",
			Name:     "ClawBench: Scientific Constant",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `What is the speed of light in a vacuum, in meters per second, rounded to the nearest million? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"300000000"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_013",
			Name:     "ClawBench: Sequence Pattern",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math"},
			Prompt:   `What is the next number in the Fibonacci sequence after 1, 1, 2, 3, 5, 8, 13, 21? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"34"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_014",
			Name:     "ClawBench: Acronym Expansion",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "factual"},
			Prompt:   `In computing, what does the acronym SQL stand for? Provide the full expansion.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Structured Query Language"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "cb_reasoning_015",
			Name:     "ClawBench: Base Conversion",
			Category: "clawbench_original",
			Tags:     []string{"clawbench", "reasoning", "math", "coding"},
			Prompt:   `What is the decimal (base 10) value of the hexadecimal number 0xFF? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"255"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
	}
}

// AllTasks returns all built-in benchmark tasks (builtin + ClawBench originals).
// GAIA official tasks are NOT included here — they're loaded at runtime
// via FetchGAIAQuestions when --hf-token is provided.
func AllTasks() []Task {
	tasks := BuiltinTasks()
	tasks = append(tasks, ClawBenchOriginalTasks()...)
	return tasks
}

// FindTask returns a task by ID from the built-in task set, or nil if not found.
func FindTask(id string) *Task {
	for _, t := range AllTasks() {
		if t.ID == id {
			return &t
		}
	}
	return nil
}
