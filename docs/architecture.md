# Architecture

## Overview

ClawBench is a single Go binary that connects to a running OpenClaw Gateway, runs benchmark tasks in isolated sessions, scores the results, and saves them as JSON. Users compare result files from different machines/setups to see which configuration performs better.

```
  Your machine                    Teammate's machine
  ┌──────────────────┐            ┌──────────────────┐
  │ OpenClaw Gateway │            │ OpenClaw Gateway │
  │ (already running)│            │ (already running)│
  │ ws://127.0.0.1:  │            │ ws://127.0.0.1:  │
  │      18789       │            │      18789       │
  └────────┬─────────┘            └────────┬─────────┘
           │                               │
  ┌────────┴─────────┐            ┌────────┴─────────┐
  │  clawbench run   │            │  clawbench run   │
  │  → results.json  │            │  → results.json  │
  └────────┬─────────┘            └────────┬─────────┘
           │                               │
           └───────────┐  ┌────────────────┘
                       v  v
              clawbench compare a.json b.json
              → terminal table with deltas
```

## File Structure

```
main.go              CLI entry point + run/compare/list/version commands
models.go            All structs: Task, RunResult, EvalResult, ConfigMeta, GatewayResponse, etc.
tasks.go             2 embedded benchmark task definitions
gateway.go           OpenClaw Gateway WebSocket client (protocol v3)
device.go            Device identity loading + Ed25519 challenge signing
cli_backend.go       CLI backend (shells out to openclaw CLI)
evaluator.go         6 built-in evaluators
runner.go            Task execution orchestrator + result aggregation
report.go            Terminal output + JSON save/load + comparison
evaluator_test.go    Evaluator unit tests
report_test.go       Report/compare/aggregation tests
```

## Backends

ClawBench supports two backends via `--mode`:

- **`cli` (default)** — Shells out to the `openclaw` CLI. Handles auth automatically since the CLI is already paired. Implemented in `cli_backend.go`.
- **`websocket`** — Direct WebSocket connection to the Gateway. Requires `--token` for auth. Loads device identity from `~/.openclaw/identity/device.json` for scope binding. Implemented in `gateway.go` + `device.go`.

Both implement the `Backend` interface: `Connect()`, `SendPrompt()`, `ServerVersion()`, `Close()`.

## Data Flow

```
1. CLI parses flags, selects backend (cli or websocket)
2. Backend.Connect()
   CLI mode:  verifies openclaw is on PATH, checks Gateway status
   WS mode:   reads connect.challenge event
              loads device identity from ~/.openclaw/identity/device.json
              sends connect req with auth + device signature + caps:["tool-events"]
              reads hello-ok response with server version
3. For each task (repeated N times if --repeat):
   a. Backend.SendPrompt()
      → sends sessions.create with prompt + unique session key
      → streams chat delta events (partial text, structured content)
      → captures agent tool events (stream: "tool", data.name)
      → waits for chat final event (complete text + usage)
   b. Evaluate() runs all evaluators against the response
   c. RunResult is assembled with scores + metadata
4. Results are aggregated (median/stddev for repeated runs)
5. BenchmarkResults written to JSON file
6. Terminal table printed
```

## Key Design Decisions

1. **Single binary, no config files.** Tasks are embedded as Go structs. No external config to discover, parse, or validate.

2. **Connect to existing Gateway.** No process lifecycle management. Users already have OpenClaw running. ClawBench just connects and benchmarks.

3. **Session isolation.** Each task creates its own session via `sessions.create` with a unique key. This prevents benchmark runs from polluting active agent conversations and ensures a clean context per task.

4. **Protocol v3 compliance.** The Gateway client implements the real OpenClaw WebSocket protocol: connect challenge/response handshake with Ed25519 device signing, typed req/res/event frames, tool-events capability, chat state machine (delta → final/error/aborted).

5. **Separate metrics, not composited.** Correctness, tool accuracy, latency, and cost are reported independently. No fake "efficiency" score that conflates different things.

6. **Graceful degradation.** If the Gateway doesn't expose token counts, cost is estimated from response length. If tool events aren't available, tool_invoked falls back to heuristic text matching. If device identity isn't found, connects without it (scopes may be limited).

7. **Config metadata in results.** Every result file captures model, Gateway version, and gateway URL so cross-machine comparisons are interpretable.
