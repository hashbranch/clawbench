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
	url        string
	authToken  string
	conn       *websocket.Conn
	mu         sync.Mutex
	reqID      int
	sessionSeq int
	serverVer  string
	debug      bool
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

	// Extract nonce from challenge for device identity signing
	nonce := ""
	if payload, ok := challengeFrame["payload"].(map[string]any); ok {
		nonce, _ = payload["nonce"].(string)
	}

	// Step 2: Build connect request with protocol version, auth, and device identity
	// ConnectIdentity ensures the connect params and device signature use identical values
	ci := DefaultConnectIdentity(c.authToken)

	connectParams := map[string]any{
		"minProtocol": protocolVersion,
		"maxProtocol": protocolVersion,
		"client": map[string]any{
			"id":          ci.ClientID,
			"displayName": "ClawBench",
			"version":     version,
			"platform":    ci.Platform,
			"mode":        ci.ClientMode,
			"instanceId":  fmt.Sprintf("clawbench-%d", time.Now().UnixMilli()),
		},
		"caps":   []string{"tool-events"},
		"role":   ci.Role,
		"scopes": strings.Split(ci.Scopes, ","),
		"auth": map[string]any{
			"token": c.authToken,
		},
	}

	// Load and attach device identity from OpenClaw CLI's stored credentials
	deviceIdentity, err := LoadDeviceIdentity()
	if err != nil {
		fmt.Printf("Warning: %s (connecting without device identity, scopes may be limited)\n", err)
	} else {
		connectParams["device"] = deviceIdentity.SignChallenge(nonce, ci)

		// Also use device token if available
		deviceAuth, _ := LoadDeviceAuth()
		if deviceAuth != nil && deviceAuth.DeviceToken != "" {
			auth := connectParams["auth"].(map[string]any)
			auth["deviceToken"] = deviceAuth.DeviceToken
		}
	}

	connectFrame := map[string]any{
		"type":   "req",
		"id":     c.nextID(),
		"method": "connect",
		"params": connectParams,
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

// SendPrompt creates an isolated session, sends the prompt, and collects the
// full response by accumulating delta events until the final signal.
func (c *GatewayClient) SendPrompt(ctx context.Context, prompt string) (GatewayResponse, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return GatewayResponse{}, fmt.Errorf("not connected")
	}

	// Create an isolated session for this benchmark task
	c.sessionSeq++
	sessionKey := fmt.Sprintf("clawbench-%d-%d", time.Now().UnixNano(), c.sessionSeq)
	reqID := c.nextID()

	createFrame := map[string]any{
		"type":   "req",
		"id":     reqID,
		"method": "sessions.create",
		"params": map[string]any{
			"key":     sessionKey,
			"agentId": "default",
			"label":   fmt.Sprintf("ClawBench - %d", c.sessionSeq),
			"message": prompt,
		},
	}
	if err := c.conn.WriteJSON(createFrame); err != nil {
		return GatewayResponse{}, fmt.Errorf("failed to send prompt: %w", err)
	}

	// Collect events. Track our runId to filter events for this session.
	var result GatewayResponse
	result.Raw = make(map[string]any)
	var textParts []string
	var allFrames []map[string]any
	var ourRunID string

	for {
		select {
		case <-ctx.Done():
			result.Text = strings.Join(textParts, "")
			return result, ctx.Err()
		default:
		}

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

		if c.debug {
			raw, _ := json.Marshal(frame)
			fmt.Printf("  [DEBUG] frame: %s\n", string(raw))
		}

		frameType, _ := frame["type"].(string)

		switch frameType {
		case "event":
			eventName, _ := frame["event"].(string)
			payload, _ := frame["payload"].(map[string]any)
			if payload == nil {
				continue
			}

			// Filter events by session key or runId
			evtSession, _ := payload["sessionKey"].(string)
			evtRunID, _ := payload["runId"].(string)

			switch eventName {
			case "chat":
				// Filter: match our session (Gateway may prefix with agent ID)
				if evtSession != "" && !strings.Contains(evtSession, sessionKey) {
					continue
				}

				// Track our runId from the first matching chat event
				if ourRunID == "" && evtRunID != "" {
					ourRunID = evtRunID
					if c.debug {
						fmt.Printf("  [DEBUG] tracking runId: %s\n", ourRunID)
					}
				}
				if ourRunID != "" && evtRunID != "" && evtRunID != ourRunID {
					continue
				}

				state, _ := payload["state"].(string)
				switch state {
				case "delta":
					if text, ok := payload["message"].(string); ok {
						textParts = append(textParts, text)
					}
				case "final":
					// Final event may or may not have message content.
					// Always prefer accumulated deltas.
					if len(textParts) > 0 {
						result.Text = strings.Join(textParts, "")
					} else if text, ok := payload["message"].(string); ok {
						result.Text = text
					}
					// Extract token usage
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
					if model, ok := payload["model"].(string); ok {
						result.Model = model
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
				// Tool lifecycle events — filter to our run
				if ourRunID != "" && evtRunID != "" && evtRunID != ourRunID {
					continue
				}
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

			case "tick", "sessions.changed":
				// Ignore housekeeping events
			}

		case "res":
			// Response to sessions.create — extract runId if available
			if payload, ok := frame["payload"].(map[string]any); ok {
				if runStarted, ok := payload["runStarted"].(bool); ok && runStarted {
					if c.debug {
						fmt.Println("  [DEBUG] session created, run started")
					}
				}
			}
			// Check for errors
			if ok, exists := frame["ok"]; exists {
				if okBool, isBool := ok.(bool); isBool && !okBool {
					errObj, _ := frame["error"].(map[string]any)
					errMsg, _ := errObj["message"].(string)
					return result, fmt.Errorf("sessions.create rejected: %s", errMsg)
				}
			}
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
