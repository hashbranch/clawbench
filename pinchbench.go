package main

import "time"

// PinchBenchTasks returns tasks adapted from the PinchBench project
// (https://github.com/pinchbench/skill). These are real-world agent tasks
// designed for OpenClaw, adapted to test how different workspace configurations
// affect performance.
//
// Original tasks benchmark models (holding config constant).
// ClawBench adapts them to benchmark configs (holding model constant).
func PinchBenchTasks() []Task {
	return []Task{
		// Sanity gate — must pass or abort
		{
			ID:       "pinch/sanity",
			Name:     "Sanity Check (PinchBench)",
			Category: "sanity",
			Tags:     []string{"pinchbench", "sanity", "gate"},
			Prompt:   `Say "Hello" and nothing else.`,
			TimeBudget: 30 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "exact_match", Patterns: []string{`(?i)hello`}, Weight: 1.0},
				{Type: "latency", Weight: 0.3},
			},
		},

		// Coding — weather script
		{
			ID:       "pinch/weather_script",
			Name:     "Weather Script (PinchBench)",
			Category: "coding",
			Tags:     []string{"pinchbench", "coding", "python", "api"},
			Prompt: `Create a Python script called weather.py that:
1. Uses the wttr.in API (no API key needed) to fetch weather for San Francisco
2. Parses the response to extract temperature, conditions, and humidity
3. Prints a formatted weather report
4. Handles errors gracefully (network issues, invalid responses)

The script should work when run with: python weather.py`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "weather.py", Weight: 1.0},
				{Type: "exact_match", Patterns: []string{`(?i)weather|temperature|wttr`}, Weight: 0.5},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// File operations — project structure
		{
			ID:       "pinch/file_structure",
			Name:     "Create Project Structure (PinchBench)",
			Category: "file_ops",
			Tags:     []string{"pinchbench", "file_ops", "project_setup"},
			Prompt: `Create a Python project structure with the following files:
1. src/main.py - A simple "Hello, World!" script
2. README.md - Basic project documentation with the project name "MyProject"
3. .gitignore - Standard Python gitignore (include __pycache__, *.pyc, venv/, .env)

Create all three files with appropriate content.`,
			TimeBudget: 90 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "src/main.py", Weight: 1.0},
				{Type: "file_exists", Path: "README.md", Weight: 1.0},
				{Type: "file_exists", Path: ".gitignore", Weight: 1.0},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// File operations — search and replace
		{
			ID:       "pinch/search_replace",
			Name:     "Search and Replace in Files (PinchBench)",
			Category: "file_ops",
			Tags:     []string{"pinchbench", "file_ops", "editing"},
			Prompt: `I have a configuration file called config.yaml in my workspace. Find all occurrences of "localhost" and replace them with "production.example.com". Also find all occurrences of port "3000" and replace with "8080". Show me what you changed.`,
			TimeBudget: 90 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "exact_match", Patterns: []string{`(?i)production\.example\.com|8080|replaced|changed`}, Weight: 1.0},
				{Type: "tool_invoked", ToolName: "exec", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// Multi-step workflow
		{
			ID:       "pinch/multi_step_workflow",
			Name:     "Multi-step API Workflow (PinchBench)",
			Category: "complex",
			Tags:     []string{"pinchbench", "complex", "multi_step", "coding"},
			Prompt: `Complete this multi-step workflow:
1. Read the file config.json in the workspace (it contains API settings)
2. Create a Python script called api_client.py that uses the settings from config.json
3. The script should define a function that makes a GET request to the URL in the config
4. Write documentation in API_DOCS.md explaining how to use the script

Create all files with working, production-quality code.`,
			TimeBudget: 180 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "api_client.py", Weight: 1.0},
				{Type: "file_exists", Path: "API_DOCS.md", Weight: 1.0},
				{Type: "tool_invoked", ToolName: "exec", Weight: 0.3},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "exact_match", Patterns: []string{`(?i)config|api|request|GET`}, Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// Comprehension — memory/fact extraction
		{
			ID:       "pinch/memory_retrieval",
			Name:     "Memory Retrieval (PinchBench)",
			Category: "comprehension",
			Tags:     []string{"pinchbench", "memory", "comprehension", "file_ops"},
			Prompt: `Read the file notes.md in your workspace. It contains meeting notes from last week. Find the date of the next quarterly review meeting and save just the date to a file called answer.txt.`,
			TimeBudget: 90 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "answer.txt", Weight: 1.0},
				{Type: "tool_invoked", ToolName: "exec", Weight: 0.3},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// Email triage (complex, multi-criteria)
		{
			ID:       "pinch/email_triage",
			Name:     "Email Inbox Triage (PinchBench)",
			Category: "organization",
			Tags:     []string{"pinchbench", "organization", "complex", "email"},
			Prompt: `Read the file inbox.md in your workspace. It contains 10 emails. Triage them by:
1. Assigning a priority (urgent/high/medium/low) to each
2. Categorizing each email (action_required/fyi/meeting/follow_up)
3. Writing a 1-sentence recommended action for each

Save your triage report to triage_report.md with a clear, structured format.`,
			TimeBudget: 180 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "triage_report.md", Weight: 1.0},
				{Type: "exact_match", Patterns: []string{
					`(?i)(urgent|high|medium|low)`,
					`(?i)(action_required|fyi|meeting|follow_up)`,
				}, Weight: 1.0},
				{Type: "tool_invoked", ToolName: "exec", Weight: 0.3},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},

		// Blog post writing
		{
			ID:       "pinch/blog_post",
			Name:     "Blog Post Writing (PinchBench)",
			Category: "writing",
			Tags:     []string{"pinchbench", "writing", "content"},
			Prompt: `Write a 500-word blog post about the benefits and challenges of remote work in 2026. Save it to blog_post.md. The post should:
1. Have a compelling title
2. Include an introduction, 3 main sections, and a conclusion
3. Be professional but engaging in tone
4. Include at least one statistic or data point (research if needed)`,
			TimeBudget: 120 * time.Second,
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "blog_post.md", Weight: 1.0},
				{Type: "exact_match", Patterns: []string{`(?i)remote work|work from home|hybrid`}, Weight: 0.5},
				{Type: "tool_invoked", ToolName: "write", Weight: 0.5},
				{Type: "cost", Weight: 0.3},
				{Type: "latency", Weight: 0.3},
			},
		},
	}
}
