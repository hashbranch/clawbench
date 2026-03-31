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
				{Type: "tool_invoked", ToolName: "weather", Weight: 1.0},
				{Type: "tool_invoked", ToolName: "file_write", Weight: 1.0},
				{Type: "file_exists", Path: "weather_haiku.txt", Weight: 1.0},
				{Type: "exact_match", Patterns: []string{`(?i)haiku|5.*7.*5|syllable`}, Weight: 0.5},
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

// FindTask returns a task by ID, or nil if not found.
func FindTask(id string) *Task {
	for _, t := range BuiltinTasks() {
		if t.ID == id {
			return &t
		}
	}
	return nil
}
