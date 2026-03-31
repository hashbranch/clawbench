# OpenClaw Gateway WebSocket Protocol (v3)

Reference for ClawBench's Gateway client implementation. Sourced from the OpenClaw repository at `src/gateway/protocol/schema/`.

## Frame Types

Three frame types, discriminated by the `type` field:

### Request (client → server)
```json
{ "type": "req", "id": "1", "method": "chat.send", "params": { ... } }
```

### Response (server → client)
```json
{ "type": "res", "id": "1", "ok": true, "payload": { ... } }
```

### Event (server → client, push)
```json
{ "type": "event", "event": "chat", "payload": { ... } }
```

## Connect Handshake

1. Server sends `connect.challenge` event with nonce
2. Client sends `connect` request:

```json
{
  "type": "req",
  "id": "1",
  "method": "connect",
  "params": {
    "minProtocol": 3,
    "maxProtocol": 3,
    "client": {
      "id": "cli",
      "displayName": "ClawBench",
      "version": "0.1.0",
      "platform": "cli",
      "mode": "cli"
    },
    "caps": ["tool-events"],
    "role": "operator",
    "scopes": ["chat", "sessions"],
    "auth": { "token": "sk-..." }
  }
}
```

3. Server responds with `hello-ok` containing server version, features, snapshot

### Auth Modes

| Mode | Description |
|------|-------------|
| `none` | No auth required |
| `token` | Token in `auth.token` field |
| `password` | Password in `auth.password` field |
| `trusted-proxy` | Tailscale/reverse proxy |

## Sending a Prompt

```json
{
  "type": "req",
  "id": "2",
  "method": "chat.send",
  "params": {
    "sessionKey": "main",
    "message": "Your prompt here",
    "idempotencyKey": "unique-key"
  }
}
```

Server acknowledges with a `res` frame, then streams results via events.

## Streaming Events

### Chat Events (`event: "chat"`)

**Delta** (streaming partial text):
```json
{ "state": "delta", "message": "partial text..." }
```

**Final** (complete response with token usage):
```json
{
  "state": "final",
  "message": "Complete response text",
  "usage": {
    "inputTokens": 150,
    "outputTokens": 320,
    "totalTokens": 620
  },
  "stopReason": "end_turn"
}
```

**Error/Aborted**:
```json
{ "state": "error", "errorMessage": "Rate limit exceeded" }
{ "state": "aborted" }
```

### Agent Events (`event: "agent"`)

Only received if `caps: ["tool-events"]` was set during connect.

**Tool invocation**:
```json
{
  "stream": "tool_use",
  "data": {
    "tool_name": "weather",
    "tool_input": { "location": "San Francisco" },
    "tool_use_id": "tu-123"
  }
}
```

### Other Events

- `tick` — heartbeat
- `shutdown` — server going down
- `sessions.changed` — session state change

## Methods Reference

| Method | Purpose |
|--------|---------|
| `connect` | Handshake (first frame only) |
| `chat.send` | Send a prompt |
| `chat.abort` | Cancel generation |
| `chat.history` | Get past messages |
| `sessions.create` | Create session + optional first message |
| `sessions.send` | Send to existing session |
| `sessions.usage` | Token/cost analytics |
| `models.list` | Available models |
| `tools.catalog` | Available tools |
| `agent.identity` | Agent info |
