package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// CLIBackend sends prompts by shelling out to `openclaw chat send`.
// This bypasses the WebSocket device identity requirement since the
// openclaw CLI is already paired with the Gateway.
type CLIBackend struct {
	sessionKey string
	serverVer  string
}

// NewCLIBackend creates a backend that uses the openclaw CLI.
func NewCLIBackend() *CLIBackend {
	return &CLIBackend{
		sessionKey: fmt.Sprintf("clawbench-%d", time.Now().UnixMilli()),
	}
}

// Connect verifies the openclaw CLI is available and the Gateway is reachable.
func (c *CLIBackend) Connect(ctx context.Context) error {
	// Check openclaw is on PATH
	_, err := exec.LookPath("openclaw")
	if err != nil {
		return fmt.Errorf("openclaw CLI not found on PATH (install OpenClaw or use --mode websocket)")
	}

	// Verify Gateway is reachable by getting status
	cmd := exec.CommandContext(ctx, "openclaw", "status", "--json")
	out, err := cmd.Output()
	if err != nil {
		// Try without --json flag
		cmd2 := exec.CommandContext(ctx, "openclaw", "status")
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			return fmt.Errorf("Gateway not reachable: %s", string(out2))
		}
		// Parse version from text output
		for _, line := range strings.Split(string(out2), "\n") {
			if strings.Contains(line, "version") || strings.Contains(line, "Version") {
				c.serverVer = strings.TrimSpace(line)
				break
			}
		}
		return nil
	}

	// Parse JSON status
	var status map[string]any
	if err := json.Unmarshal(out, &status); err == nil {
		if ver, ok := status["version"].(string); ok {
			c.serverVer = ver
		}
	}
	return nil
}

// SendPrompt shells out to `openclaw chat send` and captures the response.
func (c *CLIBackend) SendPrompt(ctx context.Context, prompt string) (GatewayResponse, error) {
	var result GatewayResponse
	result.Raw = make(map[string]any)

	// Use openclaw's send command
	// Try session-based send first, fall back to direct chat
	args := []string{"send", "--session", c.sessionKey, "--text", prompt}

	cmd := exec.CommandContext(ctx, "openclaw", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// If session-based send fails, try simpler invocation
		args2 := []string{"send", prompt}
		cmd2 := exec.CommandContext(ctx, "openclaw", args2...)
		out2, err2 := cmd2.CombinedOutput()
		if err2 != nil {
			// Last resort: try `openclaw chat` with stdin
			args3 := []string{"chat", "--once", "--no-interactive", prompt}
			cmd3 := exec.CommandContext(ctx, "openclaw", args3...)
			out3, err3 := cmd3.CombinedOutput()
			if err3 != nil {
				return result, fmt.Errorf("openclaw send failed: %s (tried 3 invocations)", strings.TrimSpace(string(out3)))
			}
			out = out3
		} else {
			out = out2
		}
	}

	result.Text = strings.TrimSpace(string(out))

	// Try to extract structured data if the output is JSON
	var jsonOut map[string]any
	if err := json.Unmarshal(out, &jsonOut); err == nil {
		if text, ok := jsonOut["text"].(string); ok {
			result.Text = text
		}
		if model, ok := jsonOut["model"].(string); ok {
			result.Model = model
		}
		if usage, ok := jsonOut["usage"].(map[string]any); ok {
			if v, ok := usage["inputTokens"].(float64); ok {
				result.Tokens.Input = int(v)
			}
			if v, ok := usage["outputTokens"].(float64); ok {
				result.Tokens.Output = int(v)
			}
			if v, ok := usage["totalTokens"].(float64); ok {
				result.Tokens.Total = int(v)
			}
			result.Tokens.Available = true
		}
		if tools, ok := jsonOut["toolCalls"].([]any); ok {
			for _, t := range tools {
				if tc, ok := t.(map[string]any); ok {
					call := ToolCall{}
					call.Name, _ = tc["name"].(string)
					call.Args, _ = tc["args"].(map[string]any)
					result.ToolCalls = append(result.ToolCalls, call)
				}
			}
		}
		result.Raw = jsonOut
	}

	return result, nil
}

// ServerVersion returns the Gateway version.
func (c *CLIBackend) ServerVersion() string {
	return c.serverVer
}

// Close is a no-op for the CLI backend.
func (c *CLIBackend) Close() error {
	return nil
}
