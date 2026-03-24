---
name: WS Enqueue Panic Fix
overview: Eliminate the WebSocket `send on closed channel` panic by introducing explicit socket shutdown coordination so producer goroutines stop before `writeCh` is closed.
todos:
  - id: add-socket-shutdown-state
    content: Add socket context, shutdown signal, and producer waitgroup to `serverSocket`.
    status: completed
  - id: guard-enqueue-and-producers
    content: Make `enqueue` and streaming loops observe shutdown/cancellation and stop safely.
    status: completed
  - id: reorder-handleconnection-shutdown
    content: Cancel/await producers before closing `writeCh` in `HandleConnection`.
    status: completed
  - id: validate-ws-panic-fix
    content: Run lint and targeted compile/tests for realtime ws packages and confirm panic path is eliminated.
    status: completed
isProject: false
---

# Fix WS `enqueue` Panic

## Problem

`internal/realtime/ws/service.go` closes `writeCh` in `HandleConnection` while producer goroutines (`runStatsStream`, logs goroutine in `startLogsStream`) may still call `enqueue`, which can panic with `send on closed channel`.

Key hotspot:

- `[/Users/florianthievent/workspace/private/spoutmc/internal/realtime/ws/service.go](/Users/florianthievent/workspace/private/spoutmc/internal/realtime/ws/service.go)`

## Root Cause

Current shutdown order in `HandleConnection`:

1. stop logs
2. close `writeCh`
3. wait writer loop

But stats/log producers are not fully synchronized to stop before step 2.

## Fix Strategy

Introduce explicit socket lifecycle coordination and make shutdown producer-safe.

### 1) Add socket-level shutdown state

In `serverSocket`:

- add `ctx context.Context` + `cancel context.CancelFunc` for socket lifetime
- add `done chan struct{}` (or equivalent closed flag) to signal shutdown
- add `wg sync.WaitGroup` for producer goroutines

### 2) Make `enqueue` shutdown-aware

- `enqueue` should return early when socket is closing (`select` on done/context)
- keep non-blocking send semantics (drop on backpressure)
- avoid sending after shutdown begins

### 3) Ensure producers are tracked and cancellable

- wrap `runStatsStream` and logs streaming goroutine with `wg.Add/Done`
- in loops, check socket context cancellation (`select` with ticker/log reads)
- set `subscribeStats` / `subscribeLogs` false on connection teardown

### 4) Reorder connection teardown safely

In `HandleConnection` after `readLoop` exits:

1. mark subscriptions false
2. call `stopLogs()` and `cancel()` socket context
3. `wg.Wait()` for producers to stop
4. close `writeCh`
5. wait writer loop closure and close websocket

This guarantees no producer can send to a closed channel.

### 5) Keep behavior parity

- Preserve existing message formats (`connected`, `stats`, `log`, `command_ack`, `error`)
- Preserve backpressure drop behavior and logging
- Preserve feature flags and auth behavior in API layer

## Validation

- Run lints on `internal/realtime/ws/service.go` and ws API wrapper file.
- Run targeted compile/tests:
  - `go test ./internal/realtime/ws ./internal/webserver/api/v1/ws`
- If possible, manually verify websocket connect/subscribe/disconnect path does not panic under rapid reconnect.

## Expected Result

No more goroutine panic from `enqueue` after disconnect; websocket shutdown is deterministic and race-safe under concurrent stats/log producers.