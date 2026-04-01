package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// mockGateway implements the OpenClaw Gateway WebSocket protocol v3.
// It handles connect handshake, sessions.create, and streams chat events
// with configurable responses.
type mockGateway struct {
	// Response config per session
	responseText string
	toolCalls    []map[string]any
	tokenUsage   map[string]any
	model        string

	// Tracking
	connectReceived bool
	sessionsCreated []string
	promptsReceived []string
}

func newMockGateway() *mockGateway {
	return &mockGateway{
		responseText: "This is a mock response.",
		model:        "mock/test-model",
		tokenUsage: map[string]any{
			"inputTokens":  float64(100),
			"outputTokens": float64(50),
			"totalTokens":  float64(150),
		},
	}
}

func (m *mockGateway) handler(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Step 1: Send connect.challenge
	conn.WriteJSON(map[string]any{
		"type":  "event",
		"event": "connect.challenge",
		"payload": map[string]any{
			"nonce": "test-nonce-12345",
			"ts":    time.Now().UnixMilli(),
		},
	})

	// Step 2: Read connect request
	_, msg, err := conn.ReadMessage()
	if err != nil {
		return
	}
	var connectReq map[string]any
	json.Unmarshal(msg, &connectReq)
	m.connectReceived = true

	reqID, _ := connectReq["id"].(string)

	// Step 3: Send hello-ok
	conn.WriteJSON(map[string]any{
		"type": "res",
		"id":   reqID,
		"ok":   true,
		"payload": map[string]any{
			"type":     "hello-ok",
			"protocol": 3,
			"server": map[string]any{
				"version": "mock-1.0.0",
				"connId":  "mock-conn-1",
			},
			"features": map[string]any{
				"methods": []string{"chat.send", "sessions.create"},
				"events":  []string{"chat", "agent"},
			},
		},
	})

	// Handle subsequent requests
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var frame map[string]any
		json.Unmarshal(msg, &frame)

		method, _ := frame["method"].(string)
		fID, _ := frame["id"].(string)
		params, _ := frame["params"].(map[string]any)

		switch method {
		case "sessions.create":
			sessionKey, _ := params["key"].(string)
			prompt, _ := params["message"].(string)
			m.sessionsCreated = append(m.sessionsCreated, sessionKey)
			m.promptsReceived = append(m.promptsReceived, prompt)

			// Acknowledge session creation
			conn.WriteJSON(map[string]any{
				"type": "res",
				"id":   fID,
				"ok":   true,
				"payload": map[string]any{
					"key":        sessionKey,
					"sessionId":  "sess-mock-1",
					"runStarted": true,
				},
			})

			runID := fmt.Sprintf("run-mock-%d", time.Now().UnixNano())

			// Send tool call events if configured
			for _, tc := range m.toolCalls {
				conn.WriteJSON(map[string]any{
					"type":  "event",
					"event": "agent",
					"payload": map[string]any{
						"runId":      runID,
						"sessionKey": sessionKey,
						"stream":     "tool",
						"data": map[string]any{
							"name":  tc["name"],
							"phase": "start",
							"input": tc["input"],
						},
					},
				})
				time.Sleep(10 * time.Millisecond)
			}

			// Send delta events (streaming text)
			words := strings.Fields(m.responseText)
			for i := 0; i < len(words); i += 3 {
				end := i + 3
				if end > len(words) {
					end = len(words)
				}
				chunk := strings.Join(words[i:end], " ")
				if i+3 < len(words) {
					chunk += " "
				}
				conn.WriteJSON(map[string]any{
					"type":  "event",
					"event": "chat",
					"payload": map[string]any{
						"runId":      runID,
						"sessionKey": sessionKey,
						"state":      "delta",
						"message":    chunk,
					},
				})
				time.Sleep(10 * time.Millisecond)
			}

			// Send final event
			conn.WriteJSON(map[string]any{
				"type":  "event",
				"event": "chat",
				"payload": map[string]any{
					"runId":      runID,
					"sessionKey": sessionKey,
					"state":      "final",
					"usage":      m.tokenUsage,
					"model":      m.model,
				},
			})
		}
	}
}

// startMockGateway creates an httptest server running the mock Gateway.
// Returns the server and its WebSocket URL.
func startMockGateway(m *mockGateway) (*httptest.Server, string) {
	server := httptest.NewServer(http.HandlerFunc(m.handler))
	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	return server, wsURL
}

// --- Integration Tests ---

func TestGatewayConnect(t *testing.T) {
	mock := newMockGateway()
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()

	err := client.Connect(ctx)
	if err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
	defer client.Close()

	if !mock.connectReceived {
		t.Error("mock did not receive connect request")
	}
	if client.ServerVersion() != "mock-1.0.0" {
		t.Errorf("expected version mock-1.0.0, got %s", client.ServerVersion())
	}
}

func TestSendPrompt_BasicResponse(t *testing.T) {
	mock := newMockGateway()
	mock.responseText = "The answer is 42."
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	resp, err := client.SendPrompt(ctx, "What is the meaning of life?")
	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}

	if resp.Text != "The answer is 42." {
		t.Errorf("expected 'The answer is 42.', got %q", resp.Text)
	}
	if len(mock.promptsReceived) != 1 {
		t.Errorf("expected 1 prompt, got %d", len(mock.promptsReceived))
	}
}

func TestSendPrompt_TokenUsage(t *testing.T) {
	mock := newMockGateway()
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	resp, err := client.SendPrompt(ctx, "test")
	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}

	if !resp.Tokens.Available {
		t.Error("expected token data to be available")
	}
	if resp.Tokens.Input != 100 {
		t.Errorf("expected 100 input tokens, got %d", resp.Tokens.Input)
	}
	if resp.Tokens.Output != 50 {
		t.Errorf("expected 50 output tokens, got %d", resp.Tokens.Output)
	}
	if resp.Tokens.Total != 150 {
		t.Errorf("expected 150 total tokens, got %d", resp.Tokens.Total)
	}
}

func TestSendPrompt_ToolCalls(t *testing.T) {
	mock := newMockGateway()
	mock.responseText = "I checked the weather. It is sunny."
	mock.toolCalls = []map[string]any{
		{"name": "exec", "input": map[string]any{"command": "curl wttr.in"}},
		{"name": "write", "input": map[string]any{"path": "haiku.txt"}},
	}
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	resp, err := client.SendPrompt(ctx, "check the weather")
	if err != nil {
		t.Fatalf("SendPrompt failed: %v", err)
	}

	if len(resp.ToolCalls) != 2 {
		t.Fatalf("expected 2 tool calls, got %d", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "exec" {
		t.Errorf("expected tool 'exec', got %q", resp.ToolCalls[0].Name)
	}
	if resp.ToolCalls[1].Name != "write" {
		t.Errorf("expected tool 'write', got %q", resp.ToolCalls[1].Name)
	}
}

func TestSendPrompt_IsolatedSessions(t *testing.T) {
	mock := newMockGateway()
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	client.SendPrompt(ctx, "first prompt")
	client.SendPrompt(ctx, "second prompt")

	if len(mock.sessionsCreated) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(mock.sessionsCreated))
	}
	if mock.sessionsCreated[0] == mock.sessionsCreated[1] {
		t.Error("sessions should have unique keys")
	}
}

func TestSendPrompt_Timeout(t *testing.T) {
	// Create a mock that never sends final
	slowMock := newMockGateway()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upgrader := websocket.Upgrader{CheckOrigin: func(r *http.Request) bool { return true }}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Send challenge
		conn.WriteJSON(map[string]any{
			"type": "event", "event": "connect.challenge",
			"payload": map[string]any{"nonce": "test", "ts": time.Now().UnixMilli()},
		})
		// Read connect, send hello-ok
		_, msg, _ := conn.ReadMessage()
		var req map[string]any
		json.Unmarshal(msg, &req)
		conn.WriteJSON(map[string]any{
			"type": "res", "id": req["id"], "ok": true,
			"payload": map[string]any{
				"type": "hello-ok", "protocol": 3,
				"server": map[string]any{"version": "mock-1.0.0"},
			},
		})
		// Read sessions.create, ack, but never send final
		_, msg2, _ := conn.ReadMessage()
		var req2 map[string]any
		json.Unmarshal(msg2, &req2)
		conn.WriteJSON(map[string]any{
			"type": "res", "id": req2["id"], "ok": true,
			"payload": map[string]any{"runStarted": true},
		})
		// Block forever
		select {}
	}))
	defer server.Close()
	_ = slowMock

	wsURL := "ws" + strings.TrimPrefix(server.URL, "http")
	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	// Use a short timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	defer cancel()

	_, err := client.SendPrompt(timeoutCtx, "this should timeout")
	if err == nil {
		t.Error("expected timeout error")
	}
	if !strings.Contains(err.Error(), "context deadline exceeded") && !strings.Contains(err.Error(), "read error") {
		t.Errorf("expected timeout-related error, got: %v", err)
	}
}

func TestRunTask_WithMockGateway(t *testing.T) {
	mock := newMockGateway()
	mock.responseText = "- Solar energy reduces emissions.\n- Wind power is cost-effective.\n- Clean energy creates jobs."
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	task := Task{
		ID:         "test_task",
		Name:       "Test Task",
		TimeBudget: 30 * time.Second,
		Evaluators: []EvalConfig{
			{Type: "exact_match", Patterns: []string{`(?i)solar|wind|energy`}, Weight: 1.0},
			{Type: "format_bullets", Weight: 1.0},
			{Type: "cost", Weight: 0.3},
			{Type: "latency", Weight: 0.3},
		},
	}

	result := RunTask(ctx, client, task, "")
	if result.IsError {
		t.Fatalf("RunTask failed: %s", result.ErrorMsg)
	}
	if result.Correctness == 0 {
		t.Error("expected non-zero correctness")
	}
	if result.WallClockSeconds == 0 {
		t.Error("expected non-zero latency")
	}
	if result.TotalTokens != 150 {
		t.Errorf("expected 150 tokens, got %d", result.TotalTokens)
	}
}

func TestRunTask_ToolDetection(t *testing.T) {
	mock := newMockGateway()
	mock.responseText = "Done. Weather checked and haiku written."
	mock.toolCalls = []map[string]any{
		{"name": "exec"},
		{"name": "write"},
	}
	server, wsURL := startMockGateway(mock)
	defer server.Close()

	client := NewGatewayClient(wsURL, "test-token")
	ctx := context.Background()
	client.Connect(ctx)
	defer client.Close()

	task := Task{
		ID:         "test_tools",
		Name:       "Test Tool Detection",
		TimeBudget: 30 * time.Second,
		Evaluators: []EvalConfig{
			{Type: "tool_invoked", ToolName: "exec", Weight: 1.0},
			{Type: "tool_invoked", ToolName: "write", Weight: 1.0},
		},
	}

	result := RunTask(ctx, client, task, "")
	if result.IsError {
		t.Fatalf("RunTask failed: %s", result.ErrorMsg)
	}
	if result.ToolAccuracy != 1.0 {
		t.Errorf("expected tool accuracy 1.0, got %f", result.ToolAccuracy)
	}
	if len(result.ToolsUsed) != 2 {
		t.Errorf("expected 2 tools used, got %d", len(result.ToolsUsed))
	}
}

func TestExecCheck_Evaluator(t *testing.T) {
	dir := t.TempDir()

	// Test passing command
	ec := EvalConfig{Type: "exec_check", Path: "echo hello", Weight: 1.0}
	result := evalExecCheck(ec, dir)
	if result.Score != 1.0 {
		t.Errorf("expected 1.0 for passing command, got %f", result.Score)
	}

	// Test failing command
	ec2 := EvalConfig{Type: "exec_check", Path: "exit 1", Weight: 1.0}
	result2 := evalExecCheck(ec2, dir)
	if result2.Score != 0 {
		t.Errorf("expected 0 for failing command, got %f", result2.Score)
	}

	// Test no workspace
	ec3 := EvalConfig{Type: "exec_check", Path: "echo test", Weight: 1.0}
	result3 := evalExecCheck(ec3, "")
	if result3.Score != 0 {
		t.Errorf("expected 0 for no workspace, got %f", result3.Score)
	}
}
