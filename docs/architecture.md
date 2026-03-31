# Architecture

## Overview

ClawBench is a single Go binary that connects to a running OpenClaw Gateway via WebSocket, runs benchmark tasks, scores the results, and saves them as JSON. Users compare result files from different machines/setups to see which configuration performs better.

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
main.go              CLI entry point + run/compare/list commands
models.go            All structs: Task, RunResult, EvalResult, ConfigMeta, etc.
tasks.go             2 embedded benchmark task definitions
gateway.go           OpenClaw Gateway WebSocket client (protocol v3)
evaluator.go         6 built-in evaluators
runner.go            Task execution orchestrator + result aggregation
report.go            Terminal output + JSON save/load + comparison
evaluator_test.go    24 evaluator unit tests
report_test.go       7 report/compare/aggregation tests
```

## Data Flow

```
1. CLI parses flags
2. GatewayClient.Connect()
   → reads connect.challenge event
   → sends connect req with auth + caps:["tool-events"]
   → reads hello-ok response with server version
3. For each task (repeated N times if --repeat):
   a. GatewayClient.SendPrompt()
      → sends chat.send req with prompt
      → streams chat delta events (partial text)
      → captures agent tool_use events (tool calls)
      → waits for chat final event (complete text + usage)
   b. Evaluate() runs all evaluators against the response
   c. RunResult is assembled with scores + metadata
4. Results are aggregated (median/stddev for repeated runs)
5. BenchmarkResults written to JSON file
6. Terminal table printed
```

## Key Design Decisions

1. **Single binary, no config files.** Tasks are embedded as Go structs. No YAML to discover, parse, or validate. Add YAML support later when community needs it.

2. **Connect to existing Gateway.** No process lifecycle management. Users already have OpenClaw running. ClawBench just connects and benchmarks.

3. **Protocol v3 compliance.** The Gateway client implements the real OpenClaw WebSocket protocol: connect challenge/response handshake, typed req/res/event frames, tool-events capability, chat state machine (delta → final/error/aborted).

4. **Separate metrics, not composited.** Correctness, tool accuracy, latency, and cost are reported independently. No fake "efficiency" score that conflates different things.

5. **Graceful degradation.** If the Gateway doesn't expose token counts, cost is estimated from response length. If tool events aren't available, tool_invoked falls back to heuristic text matching.

6. **Config metadata in results.** Every result file captures model, temperature, Gateway version, and gateway URL so cross-machine comparisons are interpretable.
