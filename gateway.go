package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Protocol version supported by this client.
const protocolVersion = 3

// GatewayClient connects to an OpenClaw Gateway via WebSocket.
type GatewayClient struct {
	url       string
	authToken string
	conn      *websocket.Conn
	mu        sync.Mutex
	reqID     int
	serverVer string
}

// NewGatewayClient creates a client for the given Gateway URL.
func NewGatewayClient(url, authToken string) *GatewayClient {
	return &GatewayClient{
		url:       url,
		authToken: authToken,
	}
}

func (c *GatewayClient) nextID() string {
	c.reqID++
	return fmt.Sprintf("%d", c.reqID)
}

// Connect establishes the WebSocket connection and authenticates.
// Follows the OpenClaw Gateway protocol v3 handshake:
// 1. Server sends connect.challenge event with nonce
// 2. Client sends connect request with auth
// 3. Server responds with hello-ok containing server info
func (c *GatewayClient) Connect(ctx context.Context) error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.DialContext(ctx, c.url, nil)
	if err != nil {
		return fmt.Errorf("cannot connect to Gateway at %s: %w (is OpenClaw running?)", c.url, err)
	}
	c.conn = conn

	// Step 1: Read the connect.challenge event from server
	c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, challengeMsg, err := c.conn.ReadMessage()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to read connect challenge: %w", err)
	}

	var challengeFrame map[string]any
	json.Unmarshal(challengeMsg, &challengeFrame)
	// Challenge is informational for device auth; we use token auth so we just need to read it

	// Step 2: Send connect request with protocol version and auth
	connectFrame := map[string]any{
		"type":   "req",
		"id":     c.nextID(),
		"method": "connect",
		"params": map[string]any{
			"minProtocol": protocolVersion,
			"maxProtocol": protocolVersion,
			"client": map[string]any{
				"id":          "cli",
				"displayName": "ClawBench",
				"version":     version,
				"platform":    "cli",
				"mode":        "cli",
				"instanceId":  fmt.Sprintf("clawbench-%d", time.Now().UnixMilli()),
			},
			"caps":  []string{"tool-events"}, // opt in to tool lifecycle events
			"role":  "operator",
			"scopes": []string{"chat", "sessions"},
			"auth": map[string]any{
				"token": c.authToken,
			},
		},
	}
	if err := c.conn.WriteJSON(connectFrame); err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to send connect frame: %w", err)
	}

	// Step 3: Read hello-ok response
	c.conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	_, helloMsg, err := c.conn.ReadMessage()
	if err != nil {
		c.conn.Close()
		return fmt.Errorf("failed to read connect response: %w", err)
	}

	var helloResp map[string]any
	if err := json.Unmarshal(helloMsg, &helloResp); err != nil {
		c.conn.Close()
		return fmt.Errorf("malformed connect response: %w", err)
	}

	// Check for error response
	if ok, exists := helloResp["ok"]; exists {
		if okBool, isBool := ok.(bool); isBool && !okBool {
			errObj, _ := helloResp["error"].(map[string]any)
			errMsg, _ := errObj["message"].(string)
			errCode, _ := errObj["code"].(string)
			c.conn.Close()
			return fmt.Errorf("authentication failed: %s (%s)", errMsg, errCode)
		}
	}

	// Extract server version from hello-ok payload
	if payload, ok := helloResp["payload"].(map[string]any); ok {
		if server, ok := payload["server"].(map[string]any); ok {
			c.serverVer, _ = server["version"].(string)
		}
	}

	// Clear read deadline for normal operation
	c.conn.SetReadDeadline(time.Time{})
	return nil
}

// SendPrompt sends a chat message via chat.send and collects the full response.
// It streams events until a "final" chat event is received.
func (c *GatewayClient) SendPrompt(ctx context.Context, prompt string) (GatewayResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return GatewayResponse{}, fmt.Errorf("not connected")
	}

	idempotencyKey := fmt.Sprintf("clawbench-%d", time.Now().UnixNano())

	// Send chat.send request
	reqFrame := map[string]any{
		"type":   "req",
		"id":     c.nextID(),
		"method": "chat.send",
		"params": map[string]any{
			"sessionKey":     "main",
			"message":        prompt,
			"idempotencyKey": idempotencyKey,
		},
	}
	if err := c.conn.WriteJSON(reqFrame); err != nil {
		return GatewayResponse{}, fmt.Errorf("failed to send prompt: %w", err)
	}

	// Collect streaming events until we get a "final" chat event
	var result GatewayResponse
	result.Raw = make(map[string]any)
	var textParts []string
	var allFrames []map[string]any

	for {
		select {
		case <-ctx.Done():
			result.Text = strings.Join(textParts, "")
			return result, ctx.Err()
		default:
		}

		// Use a reasonable read deadline to detect hangs
		c.conn.SetReadDeadline(time.Now().Add(120 * time.Second))
		_, msg, err := c.conn.ReadMessage()
		if err != nil {
			if len(textParts) > 0 {
				result.Text = strings.Join(textParts, "")
				return result, nil
			}
			return result, fmt.Errorf("read error: %w", err)
		}

		var frame map[string]any
		if err := json.Unmarshal(msg, &frame); err != nil {
			continue
		}
		allFrames = append(allFrames, frame)

		frameType, _ := frame["type"].(string)

		switch frameType {
		case "event":
			eventName, _ := frame["event"].(string)
			payload, _ := frame["payload"].(map[string]any)

			switch eventName {
			case "chat":
				state, _ := payload["state"].(string)
				switch state {
				case "delta":
					if msg, ok := payload["message"].(string); ok {
						textParts = append(textParts, msg)
					}
				case "final":
					if msg, ok := payload["message"].(string); ok {
						result.Text = msg
					} else {
						result.Text = strings.Join(textParts, "")
					}
					// Extract token usage from final event
					if usage, ok := payload["usage"].(map[string]any); ok {
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
					result.Raw["frames"] = allFrames
					return result, nil
				case "error":
					errMsg, _ := payload["errorMessage"].(string)
					result.Text = strings.Join(textParts, "")
					return result, fmt.Errorf("chat error: %s", errMsg)
				case "aborted":
					result.Text = strings.Join(textParts, "")
					return result, fmt.Errorf("chat aborted by server")
				}

			case "agent":
				// Tool lifecycle events (only received because we set caps: ["tool-events"])
				if payload != nil {
					stream, _ := payload["stream"].(string)
					data, _ := payload["data"].(map[string]any)
					if stream == "tool_use" && data != nil {
						tc := ToolCall{}
						if name, ok := data["tool_name"].(string); ok {
							tc.Name = name
						}
						if input, ok := data["tool_input"].(map[string]any); ok {
							tc.Args = input
						}
						result.ToolCalls = append(result.ToolCalls, tc)
					}
				}

			case "tick":
				// Heartbeat, ignore
			}

		case "res":
			// Response to our chat.send request (acknowledgment, not the actual chat response)
			if ok, exists := frame["ok"]; exists {
				if okBool, isBool := ok.(bool); isBool && !okBool {
					errObj, _ := frame["error"].(map[string]any)
					errMsg, _ := errObj["message"].(string)
					return result, fmt.Errorf("chat.send rejected: %s", errMsg)
				}
			}
			// The actual response comes via chat events, so continue listening
		}
	}
}

// ServerVersion returns the Gateway version obtained during connect.
func (c *GatewayClient) ServerVersion() string {
	return c.serverVer
}

// Close cleanly disconnects from the Gateway.
func (c *GatewayClient) Close() error {
	if c.conn == nil {
		return nil
	}
	return c.conn.Close()
}
