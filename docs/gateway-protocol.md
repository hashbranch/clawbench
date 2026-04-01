# OpenClaw Gateway WebSocket Protocol (v3)

Reference for ClawBench's Gateway client implementation. Sourced from the OpenClaw repository at `src/gateway/protocol/schema/`.

## Frame Types

Three frame types, discriminated by the `type` field:

### Request (client → server)
```json
{ "type": "req", "id": "1", "method": "sessions.create", "params": { ... } }
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

1. Server sends `connect.challenge` event with a `nonce`
2. Client sends `connect` request with auth, device identity, and requested scopes
3. Server responds with `hello-ok` containing server version, features, snapshot

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
      "version": "0.5.0",
      "platform": "darwin",
      "mode": "cli",
      "instanceId": "clawbench-1711900000000"
    },
    "caps": ["tool-events"],
    "role": "operator",
    "scopes": ["operator.admin", "operator.read", "operator.write", "operator.approvals", "operator.pairing"],
    "auth": {
      "token": "sk-...",
      "deviceToken": "dt-..."
    },
    "device": {
      "id": "device-uuid",
      "publicKey": "<base64url-encoded-ed25519-public-key>",
      "signature": "<base64url-encoded-ed25519-signature>",
      "signedAt": 1711900000000,
      "nonce": "<nonce-from-challenge>"
    }
  }
}
```

### Auth Modes

| Mode | Description |
|------|-------------|
| `none` | No auth required |
| `token` | Token in `auth.token` field |
| `password` | Password in `auth.password` field |
| `trusted-proxy` | Tailscale/reverse proxy |

## Device Identity

ClawBench loads the Ed25519 key pair from `~/.openclaw/identity/device.json` (the same identity used by the OpenClaw CLI). The device signature proves the client controls a paired device.

### Signature Format (v3)

The signed payload is a pipe-delimited string:

```
v3|deviceId|clientId|clientMode|role|scopes|signedAtMs|token|nonce|platform|deviceFamily
```

- `deviceId` — from `device.json`
- `clientId` — `"cli"`
- `clientMode` — `"cli"`
- `role` — `"operator"`
- `scopes` — comma-separated: `"operator.admin,operator.read,operator.write,operator.approvals,operator.pairing"`
- `signedAtMs` — Unix milliseconds
- `token` — the auth token
- `nonce` — from the `connect.challenge` event
- `platform` — runtime OS (e.g. `"darwin"`, `"linux"`)
- `deviceFamily` — empty string for ClawBench

The signature is Ed25519 over the raw bytes of this string. The Gateway reconstructs the same string from the connect params to verify.

### Key Format

- Private key: PEM-encoded PKCS8 (from `device.json`)
- Public key sent in the `device` object: raw 32-byte Ed25519 public key, base64url-encoded (not SPKI)

If device identity is not available, ClawBench connects without it but scopes may be limited.

## Sending a Prompt (sessions.create)

ClawBench uses `sessions.create` to run each task in an isolated session. This avoids interfering with active agent conversations.

```json
{
  "type": "req",
  "id": "2",
  "method": "sessions.create",
  "params": {
    "key": "clawbench-1711900000000-1",
    "agentId": "default",
    "label": "ClawBench - 1 - 1711900000000",
    "message": "Your prompt here"
  }
}
```

Server acknowledges with a `res` frame (may include `runStarted: true`), then streams results via events.

## Streaming Events

### Chat Events (`event: "chat"`)

Events are filtered by `sessionKey` (which the Gateway may prefix with agent ID) and `runId`.

**Delta** (streaming partial text):
```json
{
  "state": "delta",
  "sessionKey": "agent:main:clawbench-...",
  "runId": "run-uuid",
  "message": "partial text..."
}
```

The `message` field in deltas may be a plain string or a structured content object:
```json
{
  "message": {
    "content": [{ "type": "text", "text": "partial text..." }]
  }
}
```

ClawBench handles both formats.

**Final** (complete response with token usage):
```json
{
  "state": "final",
  "message": {
    "content": [{ "type": "text", "text": "Complete response text" }]
  },
  "usage": {
    "inputTokens": 150,
    "outputTokens": 320,
    "totalTokens": 620
  },
  "model": "anthropic/claude-sonnet-4-6",
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
  "stream": "tool",
  "data": {
    "name": "exec",
    "input": { "command": "curl -s wttr.in/SF" },
    "phase": "start"
  }
}
```

ClawBench captures tool start events (ignoring `phase: "result"`) and records the tool name from `data.name`.

Note: The code also accepts `stream: "tool_use"` with `data.tool_name` for backward compatibility, but the current Gateway uses `stream: "tool"` with `data.name`.

### Other Events

- `tick` — heartbeat
- `shutdown` — server going down
- `sessions.changed` — session state change

## Methods Reference

| Method | Purpose |
|--------|---------|
| `connect` | Handshake (first frame only) |
| `sessions.create` | Create isolated session + send first message |
| `sessions.send` | Send to existing session |
| `chat.send` | Send a prompt to an existing session (legacy) |
| `chat.abort` | Cancel generation |
| `chat.history` | Get past messages |
| `sessions.usage` | Token/cost analytics |
| `models.list` | Available models |
| `tools.catalog` | Available tools |
| `agent.identity` | Agent info |
