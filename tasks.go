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

// GAIATasks returns GAIA-inspired Level 1 benchmark tasks.
// These are modeled after the GAIA benchmark (General AI Assistants)
// by Mialon et al. (2023, arXiv:2311.12983), which tests real-world
// assistant capabilities with questions that have unambiguous factual answers.
//
// Level 1 questions require no more than ~5 steps and at most one tool.
// All questions here are text-only (no attached files required).
// Categories of questions: web search, calculation, reasoning, multi-step.
//
// Note: These are GAIA-style questions, not the original gated dataset.
// The original GAIA validation set is available (with access request) at
// https://huggingface.co/datasets/gaia-benchmark/GAIA
func GAIATasks() []Task {
	return []Task{
		{
			ID:       "gaia_l1_001",
			Name:     "GAIA L1: Simple Arithmetic",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math"},
			Prompt:   `What is the sum of the prime numbers between 20 and 40? Provide just the number as your final answer.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"120"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_002",
			Name:     "GAIA L1: Unit Conversion",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math", "conversion"},
			Prompt:   `A recipe calls for 2.5 cups of flour. If 1 cup equals 236.588 milliliters, how many liters of flour is that? Round to 2 decimal places. Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"0.59"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_003",
			Name:     "GAIA L1: Wikipedia Factual Lookup",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "web_search", "factual"},
			Prompt:   `What year was the Python programming language first released? Provide just the year.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"1991"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_004",
			Name:     "GAIA L1: Chemical Element Lookup",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "factual"},
			Prompt:   `What is the chemical symbol for the element with atomic number 74? Provide just the symbol.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"W"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_005",
			Name:     "GAIA L1: Multi-Step Calculation",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math", "multi_step"},
			Prompt:   `If you buy 3 items costing $12.50, $8.75, and $15.00, and there is a 7% sales tax applied to the total, what is the total amount you pay? Give your answer in dollars, rounded to 2 decimal places. Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"38.79"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_006",
			Name:     "GAIA L1: Historical Fact",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "web_search", "factual"},
			Prompt:   `Who was the first person to walk on the Moon, and in what year did it happen? Answer in the format: Firstname Lastname, YYYY`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Neil Armstrong, 1969"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_007",
			Name:     "GAIA L1: Reverse Lookup",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "factual"},
			Prompt:   `In the NATO phonetic alphabet, what word represents the letter 'Q'? Provide just the word.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Quebec"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_008",
			Name:     "GAIA L1: Geographic Knowledge",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "web_search", "factual"},
			Prompt:   `What is the capital city of Australia? Provide just the city name.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Canberra"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_009",
			Name:     "GAIA L1: String Manipulation",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "coding"},
			Prompt:   `How many vowels (a, e, i, o, u) are in the word "onomatopoeia"? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"8"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_010",
			Name:     "GAIA L1: Date Calculation",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math"},
			Prompt:   `If January 1, 2024 was a Monday, what day of the week was March 1, 2024? Provide just the day name (e.g., Monday).`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Friday"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_011",
			Name:     "GAIA L1: Roman Numerals",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math"},
			Prompt:   `What is the value of the Roman numeral MCMXCIV in Arabic numerals? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"1994"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_012",
			Name:     "GAIA L1: Scientific Constant",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "factual", "web_search"},
			Prompt:   `What is the speed of light in a vacuum, in meters per second, rounded to the nearest million? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"300000000"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_013",
			Name:     "GAIA L1: Sequence Pattern",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math"},
			Prompt:   `What is the next number in the Fibonacci sequence after 1, 1, 2, 3, 5, 8, 13, 21? Provide just the number.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"34"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_014",
			Name:     "GAIA L1: Acronym Expansion",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "factual", "web_search"},
			Prompt:   `In computing, what does the acronym SQL stand for? Provide the full expansion.`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "gaia_exact", Patterns: []string{"Structured Query Language"}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "gaia_l1_015",
			Name:     "GAIA L1: Base Conversion",
			Category: "gaia_l1",
			Tags:     []string{"gaia", "level1", "reasoning", "math", "coding"},
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

// AllTasks returns all benchmark tasks (builtin + GAIA).
func AllTasks() []Task {
	tasks := BuiltinTasks()
	tasks = append(tasks, GAIATasks()...)
	return tasks
}

// FindTask returns a task by ID, or nil if not found.
func FindTask(id string) *Task {
	for _, t := range AllTasks() {
		if t.ID == id {
			return &t
		}
	}
	return nil
}
