# Go Runtime/Trace Integration in ChronoGo

This document explains how ChronoGo integrates with Go's native `runtime/trace` package to provide automatic recording of concurrency events without manual instrumentation.

## Overview

While ChronoGo provides manual instrumentation functions for recording concurrency events (goroutine creation, channel operations, etc.), a more practical approach for real-world applications is to leverage Go's built-in tracing facilities. The `runtime/trace` package provides a way to capture and record execution traces of Go programs.

The integration in ChronoGo:
1. Uses Go's `runtime/trace` package to generate trace data
2. Maps runtime trace events to ChronoGo's event model
3. Provides transparent capture of goroutine, channel, and mutex operations
4. Produces both ChronoGo event logs and standard Go trace files

## Implementation Details

### Trace Events Capture

ChronoGo's integration with `runtime/trace` is implemented in `pkg/instrumentation/runtime_trace.go`. The key components are:

1. **Initialization**: `InitRuntimeTracing()` sets up the integration with both ChronoGo's recorder and Go's runtime/trace.

2. **Goroutine Tracking**: Using `runtime.Stack()` to periodically scan for goroutines and track their states.

3. **Runtime Metadata Mapping**: Mapping Go's internal IDs (goroutine IDs, channel addresses, etc.) to ChronoGo's sequential IDs.

4. **API for External Code**: Providing `TraceChannelOperation()` and `TraceMutexOperation()` for explicit instrumentation when needed.

### Challenges and Solutions

**Goroutine ID Extraction**: Go doesn't export goroutine IDs in a public API. ChronoGo extracts them from stack traces using `runtime.Stack()`.

**Tracking Runtime Objects**: Channels and mutexes don't have public IDs, so ChronoGo uses pointer addresses and maintains mappings to stable sequential IDs.

**Combining Trace Data**: ChronoGo maintains both native Go trace output (in `chrono_trace.out`) and ChronoGo's own event format.

## Usage

To use runtime/trace integration:

```go
import (
    "github.com/willibrandon/ChronoGo/pkg/instrumentation"
    "github.com/willibrandon/ChronoGo/pkg/recorder"
)

func main() {
    // Create a recorder
    rec := recorder.NewInMemoryRecorder()
    
    // Initialize runtime trace integration
    err := instrumentation.InitRuntimeTracing(rec)
    if err != nil {
        // Handle error
    }
    defer instrumentation.StopRuntimeTracing()
    
    // Run your concurrent program as normal
    // All concurrency events will be automatically tracked
    
    // For explicit tracing (when automatic detection fails):
    ch := make(chan int)
    instrumentation.TraceChannelOperation(ch, "send", 42)
    ch <- 42
}
```

## Benefits Over Manual Instrumentation

1. **Reduced Boilerplate**: No need to manually add instrumentation calls for each goroutine or channel operation.

2. **Complete Coverage**: Captures all goroutines, not just those explicitly instrumented.

3. **Standard Tools**: The generated `chrono_trace.out` file can be analyzed with standard Go tools (`go tool trace`).

4. **Performance**: Better performance as some instrumentation happens at the runtime level.

## Limitations

1. **Goroutine ID Stability**: The goroutine ID extraction is based on parsing stack traces, which is not an official API.

2. **Completeness**: Some low-level runtime events may not be captured perfectly.

3. **Overhead**: While more efficient than manual instrumentation, it still adds performance overhead.

## Integration with Time-Travel Debugging

The runtime/trace integration complements ChronoGo's time-travel debugging by providing more complete information about concurrency events. The recorded events can be replayed in ChronoGo's replayer just like manually instrumented events.

## Future Improvements

1. **Custom Runtime Hooks**: Consider deeper runtime integration through custom Go builds with explicit hooks.

2. **Delve Integration**: Combine runtime/trace with Delve debugger for better correlation with debug states.

3. **Visual Timeline**: Provide a visual timeline of concurrent execution combining both runtime/trace data and ChronoGo events. 