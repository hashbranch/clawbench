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

// RegressionTasks returns tasks that encode real agent mistakes.
// These are based on actual failures observed in production use,
// designed to prevent regressions in professional communication,
// email handling, and classification accuracy.
func RegressionTasks() []Task {
	return []Task{
		{
			ID:       "reg_email_draft_composition",
			Name:     "Regression: Email Draft Composition",
			Category: "regression",
			Tags:     []string{"regression", "email", "draft", "professional"},
			Prompt: `I need to reply to an email thread with James Rivera (james@riverarealty.com). His assistant Sarah (sarah@riverarealty.com) proposed a meeting time. CC Alex Chen (alex@acmeconsulting.com). Confirm the meeting and ask James to fill out a form at https://example.com/intake. Create a draft, don't send it.`,
			TimeBudget: 90 * time.Second,
			Evaluators: []EvalConfig{
				// Must mention creating a draft (not sending)
				{Type: "exact_match", Patterns: []string{`(?i)\bdraft\b`}, Weight: 1.0},
				// Must include all three recipients
				{Type: "exact_match", Patterns: []string{
					`(?i)james@riverarealty\.com`,
					`(?i)sarah@riverarealty\.com`,
					`(?i)alex@acmeconsulting\.com`,
				}, Weight: 1.5},
				// Must include the form link
				{Type: "exact_match", Patterns: []string{`https://example\.com/intake`}, Weight: 1.0},
				// Must NOT contain em dashes (AI giveaway)
				{Type: "regex_reject", Patterns: []string{`\x{2014}`, `\x{2013}`}, Weight: 1.0},
				// Must NOT say "sent" or "sending" (should be draft only)
				{Type: "regex_reject", Patterns: []string{`(?i)\b(sent the email|email has been sent|sending the email|I('ve| have) sent)\b`}, Weight: 1.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "reg_email_triage_classification",
			Name:     "Regression: Email Triage Classification",
			Category: "regression",
			Tags:     []string{"regression", "email", "triage", "classification"},
			Prompt: `Classify these emails as 'action' (needs response), 'todo' (important, not urgent), or 'noise' (archive). Reply with ONLY the classification for each, numbered.

1. From: noreply@crmsystem.com — Subject: 'New activity assigned to you'
2. From: counsel@lawfirm.com — Subject: 'Re: Vendor agreement review'
3. From: hello@localbar.com — Subject: '4/5 Free Wine Tasting Party!'
4. From: cofounder@yourcompany.com — Subject: 'Quick question about the investor deck'
5. From: notifications@github.com — Subject: 'PR #47 merged'
6. From: partner@clientcorp.com — Subject: 'Contract ready for signature'
7. From: newsletter@beehiiv.com — Subject: 'Hire Slow, Fire Fast'
8. From: assistant@prospectco.com — Subject: 'Re: Meeting reschedule'`,
			TimeBudget: 60 * time.Second,
			Evaluators: []EvalConfig{
				// Pipe response to validation script (checks 7/8 correct minimum)
				{Type: "response_check", Path: "scripts/check_email_triage.sh", Weight: 2.0},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
		{
			ID:       "reg_professional_tone_compliance",
			Name:     "Regression: Professional Tone Compliance",
			Category: "regression",
			Tags:     []string{"regression", "email", "tone", "professional"},
			Prompt: `Write a short email to a potential consulting client named James who is a commercial real estate broker. We met at a networking event. Follow up to schedule an initial call about how AI automation could help his brokerage workflows. Keep it under 100 words.`,
			TimeBudget: 60 * time.Second,
			Evaluators: []EvalConfig{
				// Pipe response to comprehensive tone validation script
				{Type: "response_check", Path: "scripts/check_professional_tone.sh", Weight: 2.0},
				// Must NOT contain em dashes
				{Type: "regex_reject", Patterns: []string{`\x{2014}`, `\x{2013}`}, Weight: 1.0},
				// Must NOT contain sycophantic openers
				{Type: "regex_reject", Patterns: []string{
					`(?i)I'd be happy to`,
					`(?i)^Great to`,
					`(?i)^Absolutely`,
					`(?i)I hope this email finds you well`,
				}, Weight: 1.0},
				// Must NOT contain buzzwords
				{Type: "regex_reject", Patterns: []string{`(?i)\b(leverage|synergy|synergies)\b`}, Weight: 0.5},
				// Must contain scheduling language
				{Type: "exact_match", Patterns: []string{`(?i)(schedule|call|meeting|chat|connect|book|set up)`}, Weight: 1.0},
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
	tasks = append(tasks, RegressionTasks()...)
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
