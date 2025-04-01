# Concurrency Instrumentation in ChronoGo

This document explains the concurrency instrumentation features added to ChronoGo to enable time-travel debugging for concurrent Go programs.

## Overview

Concurrency is one of Go's key strengths, but debugging concurrent programs can be challenging due to non-deterministic behavior. ChronoGo's concurrency instrumentation addresses this by:

1. Recording goroutine creation and switching events
2. Tracking channel operations (send, receive, close)
3. Monitoring synchronization primitives (e.g., mutexes)
4. Enabling deterministic replay of concurrent execution

## Implementation Details

### Event Types

Two new event types have been added to the recorder system:

- `ChannelOperation`: Records channel sends, receives, and closes
- `SyncOperation`: Records mutex locks and unlocks

### Runtime Hooks

In a production implementation, these functions would be injected via runtime hooks or a custom patched Go runtime. For demonstration purposes, we've implemented them as explicit function calls in `pkg/instrumentation/concurrency.go`:

- `GoroutineCreate(gID int)`: Records goroutine creation
- `GoroutineSwitch(fromID, toID int)`: Records scheduler switching between goroutines
- `ChannelSend(chID, senderID int, data interface{})`: Records channel send operations
- `ChannelRecv(chID, receiverID int, data interface{})`: Records channel receive operations
- `ChannelClose(chID, goroutineID int)`: Records channel close operations
- `MutexLock(mutexID, goroutineID int)`: Records mutex lock acquisitions
- `MutexUnlock(mutexID, goroutineID int)`: Records mutex unlock operations

### Replayer Enhancements

The replayer has been enhanced to track goroutine and channel states during replay:

1. `GoroutineState`: Tracks the state of each goroutine (ID, running status)
2. `ChannelState`: Tracks the state of each channel (ID, message queue, closed status)
3. `processGoroutineAndChannelEvents`: Processes concurrency events to update internal state

## Integration with Runtime

For full production use, ChronoGo would integrate with Go's runtime in one of these ways:

1. **Runtime Trace Package**: Using Go's built-in [runtime/trace](https://pkg.go.dev/runtime/trace) package to hook into scheduling events
2. **Custom Runtime Patch**: Patching the Go runtime to inject instrumentation at key points
3. **Delve Integration**: Using Delve's runtime hooks to capture goroutine and channel operations

## Usage Example

A demonstration program is provided in `examples/concurrency_demo.go`. For production use, you would not need to manually call the instrumentation functions - they would be automatically injected.

## Deterministic Replay Challenges

True deterministic replay of concurrent programs presents several challenges:

1. **Scheduler Non-determinism**: Go's scheduler makes non-deterministic decisions
2. **External Influences**: I/O, timers, and other external factors affect scheduling
3. **Race Conditions**: Data races can cause different results across runs

ChronoGo addresses these by recording the *actual* sequence of concurrent events that occurred, allowing exact replay of that specific execution path.

## Future Improvements

1. **Automated Runtime Integration**: Automatic injection of hooks without manual instrumentation
2. **Race Detector Integration**: Combining with Go's race detector for enhanced debugging
3. **Advanced Visualization**: Visual representation of goroutine scheduling and channel operations
4. **State Modification**: Allow modifying goroutine scheduling during replay for "what-if" scenarios

## Testing

Unit tests for concurrency instrumentation are available in `tests/concurrency_test.go`. 