package main

import (
	"context"
	"time"
)

// SetupFile defines a file to create in the workspace before running a task.
type SetupFile struct {
	Path    string // relative path in workspace
	Content string // file content
}

// Task defines a benchmark task with its prompt and evaluation criteria.
type Task struct {
	ID          string
	Name        string
	Category    string
	Tags        []string
	Prompt      string
	TimeBudget  time.Duration
	Evaluators  []EvalConfig
	SetupFiles  []SetupFile // files to seed in workspace before running
}

// EvalConfig is a flat struct for all evaluator types.
// Type is the discriminator. Unused fields are zero-value.
type EvalConfig struct {
	Type     string   // "exact_match", "tool_invoked", "file_exists", "cost", "latency"
	Patterns []string // for exact_match: regex patterns
	ToolName string   // for tool_invoked: expected tool name
	Path     string   // for file_exists: expected file path
	Weight   float64  // relative weight for scoring
}

// EvalResult is the outcome of a single evaluator.
type EvalResult struct {
	Type    string  `json:"type"`
	Score   float64 `json:"score"`   // 0.0 to 1.0
	Weight  float64 `json:"weight"`
	Passed  bool    `json:"passed"`
	Details string  `json:"details"`
}

// RunResult captures everything about a single task run.
type RunResult struct {
	RunID     string    `json:"run_id"`
	TaskID    string    `json:"task_id"`
	Timestamp time.Time `json:"timestamp"`

	// Scores (separate metrics, not composited)
	Correctness  float64 `json:"correctness"`
	ToolAccuracy float64 `json:"tool_accuracy"`

	// Raw metrics
	WallClockSeconds float64  `json:"wall_clock_seconds"`
	TotalTokens      int      `json:"total_tokens"`
	CostUSD          float64  `json:"cost_usd"`
	NumToolCalls     int      `json:"num_tool_calls"`
	ToolsUsed        []string `json:"tools_used"`

	// Metadata about the setup being benchmarked
	Config ConfigMeta `json:"config"`

	// Detailed
	EvalResults []EvalResult `json:"eval_results"`
	RawResponse string       `json:"raw_response"`
	IsError     bool         `json:"is_error"`
	ErrorMsg    string       `json:"error_message,omitempty"`
}

// ConfigMeta captures what configuration was being benchmarked.
// Without this, cross-machine comparison is meaningless.
type ConfigMeta struct {
	Label       string `json:"label"`                 // user-provided label for this run
	Model       string `json:"model,omitempty"`        // model name from Gateway
	Temperature float64 `json:"temperature,omitempty"` // sampling temperature if available
	GatewayURL  string `json:"gateway_url"`
	GatewayVer  string `json:"gateway_version,omitempty"`
	WorkspaceHash string `json:"workspace_hash,omitempty"` // hash of SOUL.md + AGENTS.md + skills
}

// BenchmarkResults is the top-level structure written to the results JSON file.
type BenchmarkResults struct {
	Version   string      `json:"version"`
	Timestamp time.Time   `json:"timestamp"`
	Label     string      `json:"label"`
	Config    ConfigMeta  `json:"config"`
	Results   []RunResult `json:"results"`
	Summary   Summary     `json:"summary"`
}

// Summary aggregates across all task runs (supports --repeat N with median/stddev).
type Summary struct {
	TotalTasks      int     `json:"total_tasks"`
	TotalRuns       int     `json:"total_runs"`
	RepeatCount     int     `json:"repeat_count"`
	AvgCorrectness  float64 `json:"avg_correctness"`
	AvgLatency      float64 `json:"avg_latency_seconds"`
	TotalCost       float64 `json:"total_cost_usd"`
	TotalTokens     int     `json:"total_tokens"`
	ScorePerKTokens float64 `json:"score_per_1k_tokens,omitempty"` // efficiency: correctness per 1K tokens
	ScorePerDollar  float64 `json:"score_per_dollar,omitempty"`    // efficiency: correctness per dollar
}

// Backend is the interface for sending prompts to OpenClaw.
// Implemented by GatewayClient (WebSocket) and CLIBackend (openclaw CLI).
type Backend interface {
	Connect(ctx context.Context) error
	SendPrompt(ctx context.Context, prompt string) (GatewayResponse, error)
	ServerVersion() string
	Close() error
}

// GatewayResponse is what we get back from the OpenClaw Gateway.
// Fields are populated based on what the protocol actually returns.
// Unknown fields are zero-value (graceful degradation).
type GatewayResponse struct {
	Text       string            // final response text
	ToolCalls  []ToolCall        // tool invocations observed
	Tokens     TokenUsage        // token counts if available
	Model      string            // model that served the response
	Temperature float64          // sampling temperature if reported
	Duration   time.Duration     // server-reported duration if available
	Raw        map[string]any    // raw protocol frames for trace capture
}

// ToolCall represents a single tool invocation observed in the Gateway response.
type ToolCall struct {
	Name   string         `json:"name"`
	Args   map[string]any `json:"args,omitempty"`
	Result string         `json:"result,omitempty"`
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	Input    int `json:"input"`
	Output   int `json:"output"`
	Total    int `json:"total"`
	Available bool `json:"available"` // false if Gateway didn't report tokens
}
