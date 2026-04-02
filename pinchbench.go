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
			SetupFiles: []SetupFile{
				{Path: "config.yaml", Content: `# Application Configuration
server:
  host: localhost
  port: 3000
  workers: 4

database:
  host: localhost
  port: 5432
  name: myapp_dev

redis:
  host: localhost
  port: 6379

api:
  base_url: http://localhost:3000/api/v1
  timeout: 30

logging:
  level: debug
  output: stdout
`},
			},
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
			SetupFiles: []SetupFile{
				{Path: "config.json", Content: `{
  "api": {
    "base_url": "https://jsonplaceholder.typicode.com",
    "endpoints": {
      "posts": "/posts",
      "users": "/users"
    },
    "timeout": 30,
    "retries": 3
  }
}
`},
			},
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
			SetupFiles: []SetupFile{
				{Path: "notes.md", Content: `# Meeting Notes - Week of March 24, 2026

## Monday Standup
- Sprint velocity looking good at 42 points
- Deploy scheduled for Wednesday
- Sarah mentioned the design review is pushed to next sprint

## Wednesday All-Hands
- Q1 revenue targets exceeded by 12%
- New hire starting April 7th (backend engineer)
- The next quarterly review meeting is scheduled for April 15, 2026
- Office renovation starts May 1st

## Friday Retro
- Improved CI pipeline reduced build times by 40%
- Need to address flaky integration tests
- Team lunch planned for next Tuesday
`},
			},
			Evaluators: []EvalConfig{
				{Type: "file_exists", Path: "answer.txt", Weight: 1.0},
				{Type: "gaia_exact", Patterns: []string{"April 15, 2026"}, Weight: 1.0},
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
			SetupFiles: []SetupFile{
				{Path: "inbox.md", Content: `# Inbox

## Email 1: Production Database Alert
From: monitoring@ops.internal
Subject: CRITICAL: Database CPU at 95%
Time: 9:02 AM

Production database server db-primary-01 has sustained CPU usage above 95% for the past 15 minutes. Active connections: 847 (normal: ~200). Slow query log shows multiple full table scans on the orders table.

## Email 2: Team Lunch Tomorrow
From: sarah@company.com
Subject: Lunch at Nopa tomorrow?
Time: 9:15 AM

Hey team! Want to do lunch at Nopa tomorrow around 12:30? They have great vegetarian options. Let me know if you can make it!

## Email 3: Client Contract Renewal
From: legal@company.com
Subject: Acme Corp contract expires Friday
Time: 9:30 AM

The Acme Corp enterprise contract (ARR $480K) expires this Friday. They've indicated interest in renewing but want to discuss pricing. Sales needs legal review of the amended terms by Thursday EOD.

## Email 4: Sprint Planning Reminder
From: jira@atlassian.com
Subject: Sprint 24 planning starts in 1 hour
Time: 10:00 AM

Sprint 24 planning ceremony begins at 11:00 AM in the Willow conference room. Please have your backlog items estimated before the meeting.

## Email 5: Security Vulnerability Report
From: security@company.com
Subject: CVE-2026-1234 affects our auth library
Time: 10:15 AM

A critical CVE has been published for auth-jwt v3.2.1 which we use in production. The vulnerability allows token forgery. A patched version (v3.2.2) is available. Please prioritize upgrading.

## Email 6: Weekly Metrics Dashboard
From: analytics@company.com
Subject: Weekly KPI Report - Week 13
Time: 10:30 AM

Weekly metrics are ready. Key highlights: DAU up 8% WoW, conversion rate steady at 3.2%, churn decreased to 1.1%. Full dashboard: https://analytics.internal/weekly

## Email 7: PTO Request Approval
From: hr@company.com
Subject: PTO request from Mike Chen needs your approval
Time: 11:00 AM

Mike Chen has requested PTO for April 21-25 (5 days). His current balance is 12 days. Please approve or deny in the HR portal by end of week.

## Email 8: Investor Update Draft
From: ceo@company.com
Subject: Review investor update before send
Time: 11:30 AM

Please review the attached Q1 investor update before I send it out tomorrow. Particularly need eyes on the revenue projections section and the new product roadmap slide. Comments by 5 PM today.

## Email 9: New Feature Request
From: product@company.com
Subject: Feature request: Bulk export to CSV
Time: 12:00 PM

Multiple enterprise customers have requested bulk data export to CSV. Currently they can only export 100 rows at a time. Proposing we add this to the Sprint 25 backlog. Product spec draft attached.

## Email 10: Office WiFi Issues
From: it@company.com
Subject: Known issue: 5GHz WiFi intermittent on Floor 3
Time: 12:30 PM

We're aware of connectivity issues on the 5GHz band on Floor 3. A replacement access point has been ordered and should arrive Thursday. In the meantime, please use the 2.4GHz network (CompanyNet-Legacy).
`},
			},
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
			TimeBudget: 240 * time.Second,
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
