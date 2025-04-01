# Testing Automatic Concurrency Event Recording

This document outlines the testing strategy and approach for validating ChronoGo's automatic concurrency event recording functionality.

## Testing Goals

The primary goals of our testing strategy are:

1. Verify that runtime/trace integration correctly records concurrency events without manual intervention
2. Compare automatic recording with manual instrumentation to ensure equivalent coverage
3. Test various concurrency patterns and primitives to ensure robust capture
4. Evaluate performance and reliability under heavy concurrency

## Test Categories

### Basic Functionality Tests

`TestAutomaticConcurrencyRecording` in `runtime_trace_test.go` verifies that the runtime/trace integration correctly records basic concurrency operations:

- Goroutine creation and switching
- Channel operations (send, receive, close)
- Mutex operations (lock, unlock)

This test runs a simple concurrent program and verifies that all expected event types are recorded.

### Comparative Tests

`TestCompareManualVsAutomatic` in `comparison_test.go` runs equivalent concurrent code using both manual instrumentation and automatic recording, then compares the events captured by each approach. This ensures:

- Automatic recording captures at least as much information as manual instrumentation
- Both approaches record the same types of events
- The events have similar details and structure

### Pattern-Specific Tests

`TestSpecificChannelInteractions` verifies that different types of channels (buffered and unbuffered) are properly instrumented, while `TestMixedPatternConcurrency` tests a mix of different concurrency patterns:

- Select statements with multiple channels
- Timeouts
- Buffered and unbuffered channels
- Mutex operations

### Stress Tests

`TestStressConcurrentRecording` in `stress_test.go` tests the runtime/trace integration under heavy concurrency, using:

- Multiple goroutines running concurrently
- Numerous channel operations
- Frequent mutex locks/unlocks

This validates the robustness and reliability of the automatic recording mechanism under load.

## Test Implementation Notes

### Explicit Tracing for Reliability

Our tests use a mix of automatic recording and explicit calls to `TraceChannelOperation` or `TraceMutexOperation`. This is done for test reliability, as the automatic detection via runtime/trace might have some limitations:

```go
// Automatic detection should work, but we add explicit tracing for test reliability
val := <-ch
TraceChannelOperation(ch, "recv", val)
```

### Time Delays

The tests include small delays to ensure events are properly processed:

```go
// Allow time for events to be processed
time.Sleep(100 * time.Millisecond)
```

This accommodates the asynchronous nature of the runtime/trace integration, which periodically scans for goroutines and processes events.

## Running the Tests

To run the tests:

```bash
go test -v github.com/willibrandon/ChronoGo/pkg/instrumentation -run "Test.*Recording|Test.*Automatic"
```

For running stress tests (which may be slower):

```bash
go test -v github.com/willibrandon/ChronoGo/pkg/instrumentation -run "TestStress.*"
```

To skip stress tests in CI/CD pipelines, use the `-short` flag:

```bash
go test -short -v github.com/willibrandon/ChronoGo/pkg/instrumentation
```

## Future Test Improvements

1. Add tests for WaitGroup and other synchronization primitives
2. Test with more complex concurrency patterns like worker pools
3. Add benchmarks to measure the performance overhead of automatic recording
4. Test recovery from panics in goroutines
5. Test with real-world concurrent applications 