package instrumentation

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"runtime/trace"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// traceIntegration maintains the state for runtime/trace integration
type traceIntegration struct {
	recorder        recorder.Recorder
	goroutineMap    sync.Map // Maps runtime goroutine IDs to our sequential IDs
	channelMap      sync.Map // Maps channel pointers to our sequential IDs
	mutexMap        sync.Map // Maps mutex pointers to our sequential IDs
	nextGoroutineID int32
	nextChannelID   int32
	nextMutexID     int32
	ctx             context.Context
	cancel          context.CancelFunc
}

var (
	traceInt      *traceIntegration
	traceInitOnce sync.Once
)

// InitRuntimeTracing initializes runtime/trace integration
func InitRuntimeTracing(rec recorder.Recorder) error {
	// Reset the initialization flag so each test can initialize its own instance
	traceInitOnce = sync.Once{}

	var initErr error

	traceInitOnce.Do(func() {
		// Create the integration state
		ctx, cancel := context.WithCancel(context.Background())
		traceInt = &traceIntegration{
			recorder:        rec,
			nextGoroutineID: 1, // Start at 1 (main goroutine)
			nextChannelID:   1,
			nextMutexID:     1,
			ctx:             ctx,
			cancel:          cancel,
		}

		// Store the main goroutine mapping
		mainGID := getGoroutineID()
		traceInt.goroutineMap.Store(mainGID, 1)

		// Create a unique filename for trace output
		traceFileName := fmt.Sprintf("chrono_trace_%d.out", time.Now().UnixNano())

		// Start the trace
		f, err := os.Create(traceFileName)
		if err != nil {
			initErr = fmt.Errorf("failed to create trace output file: %v", err)
			return
		}

		if err := trace.Start(f); err != nil {
			f.Close()
			initErr = fmt.Errorf("failed to start runtime tracing: %v", err)
			return
		}

		// Initialize our global recorder for manual instrumentation
		InitInstrumentation(rec)

		// Start a goroutine that periodically checks for new goroutines
		go monitorGoroutines(ctx)
	})

	return initErr
}

// StopRuntimeTracing stops runtime trace integration
func StopRuntimeTracing() {
	if traceInt != nil && traceInt.cancel != nil {
		traceInt.cancel()
		trace.Stop()
	}
}

// monitorGoroutines periodically scans for new goroutines using runtime.Stack()
func monitorGoroutines(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			captureGoroutineInfo()
		}
	}
}

// captureGoroutineInfo captures information about running goroutines
func captureGoroutineInfo() {
	// Get stack trace of all goroutines
	buf := make([]byte, 1<<20)
	n := runtime.Stack(buf, true)
	stacks := string(buf[:n])

	// Parse goroutine info from stack trace
	for _, stack := range strings.Split(stacks, "\n\n") {
		if strings.HasPrefix(stack, "goroutine ") {
			parseGoroutineStack(stack)
		}
	}
}

// parseGoroutineStack extracts goroutine information from a stack trace
func parseGoroutineStack(stack string) {
	// Get the goroutine ID from the first line, format: "goroutine 1 [running]:"
	firstLine := strings.Split(stack, "\n")[0]
	parts := strings.Fields(firstLine)
	if len(parts) < 2 {
		return
	}

	runtimeGID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return
	}

	// Get our internal goroutine ID or assign a new one
	var ourGID int32
	if val, ok := traceInt.goroutineMap.Load(runtimeGID); ok {
		// Make sure we get an int32, not interface{} cast to int32
		switch v := val.(type) {
		case int32:
			ourGID = v
		case int:
			ourGID = int32(v)
		case int64:
			ourGID = int32(v)
		default:
			// If it's some other type, assign a new ID to be safe
			// This shouldn't normally happen
			fmt.Printf("Warning: Unexpected type in goroutine map: %T\n", val)
			ourGID = traceInt.nextGoroutineID
			traceInt.nextGoroutineID++
			traceInt.goroutineMap.Store(runtimeGID, ourGID)
		}
	} else {
		ourGID = traceInt.nextGoroutineID
		traceInt.nextGoroutineID++
		traceInt.goroutineMap.Store(runtimeGID, ourGID)

		// Record the new goroutine creation
		GoroutineCreate(int(ourGID))
	}

	// Extract the goroutine state (running, waiting, etc.)
	state := "unknown"
	if len(parts) >= 3 {
		state = strings.Trim(parts[2], "[]:")
	}

	// Record state changes for significant states
	if state == "running" || state == "waiting" || state == "locked" {
		if traceInt.recorder != nil {
			traceInt.recorder.RecordEvent(recorder.Event{
				ID:        time.Now().UnixNano(),
				Timestamp: time.Now(),
				Type:      recorder.GoroutineSwitch,
				Details:   fmt.Sprintf("Goroutine %d state: %s", ourGID, state),
			})
		}
	}
}

// TraceChannelOperation records a channel operation using our instrumentation and runtime trace
func TraceChannelOperation(ch interface{}, op string, value interface{}) {
	if traceInt == nil {
		return
	}

	// Get channel ID from pointer
	chPtr := fmt.Sprintf("%p", ch)

	var chID int32
	if val, ok := traceInt.channelMap.Load(chPtr); ok {
		chID = val.(int32)
	} else {
		chID = traceInt.nextChannelID
		traceInt.nextChannelID++
		traceInt.channelMap.Store(chPtr, chID)

		// Record channel creation
		if traceInt.recorder != nil {
			traceInt.recorder.RecordEvent(recorder.Event{
				ID:        time.Now().UnixNano(),
				Timestamp: time.Now(),
				Type:      recorder.ChannelOperation,
				Details:   fmt.Sprintf("Channel %d created", chID),
			})
		}
	}

	// Get goroutine ID
	gID := int(getGoroutineIDOrAssign())

	// Record the operation
	switch op {
	case "send":
		// Create a trace region for this operation
		ctx, task := trace.NewTask(context.Background(), "channelSend")
		defer task.End()
		trace.Log(ctx, "channelID", fmt.Sprintf("%d", chID))
		trace.Log(ctx, "goroutineID", fmt.Sprintf("%d", gID))
		trace.Log(ctx, "value", fmt.Sprintf("%v", value))

		// Record using our instrumentation
		ChannelSend(int(chID), gID, value)

	case "recv":
		ctx, task := trace.NewTask(context.Background(), "channelRecv")
		defer task.End()
		trace.Log(ctx, "channelID", fmt.Sprintf("%d", chID))
		trace.Log(ctx, "goroutineID", fmt.Sprintf("%d", gID))
		trace.Log(ctx, "value", fmt.Sprintf("%v", value))

		// Record using our instrumentation
		ChannelRecv(int(chID), gID, value)

	case "close":
		ctx, task := trace.NewTask(context.Background(), "channelClose")
		defer task.End()
		trace.Log(ctx, "channelID", fmt.Sprintf("%d", chID))
		trace.Log(ctx, "goroutineID", fmt.Sprintf("%d", gID))

		// Record using our instrumentation
		ChannelClose(int(chID), gID)
	}
}

// TraceMutexOperation records a mutex operation using our instrumentation and runtime trace
func TraceMutexOperation(mu interface{}, op string) {
	if traceInt == nil {
		return
	}

	// Get mutex ID from pointer
	muPtr := fmt.Sprintf("%p", mu)

	var muID int32
	if val, ok := traceInt.mutexMap.Load(muPtr); ok {
		muID = val.(int32)
	} else {
		muID = traceInt.nextMutexID
		traceInt.nextMutexID++
		traceInt.mutexMap.Store(muPtr, muID)
	}

	// Get goroutine ID
	gID := int(getGoroutineIDOrAssign())

	// Record the operation
	switch op {
	case "lock":
		ctx, task := trace.NewTask(context.Background(), "mutexLock")
		defer task.End()
		trace.Log(ctx, "mutexID", fmt.Sprintf("%d", muID))
		trace.Log(ctx, "goroutineID", fmt.Sprintf("%d", gID))

		// Record using our instrumentation
		MutexLock(int(muID), gID)

	case "unlock":
		ctx, task := trace.NewTask(context.Background(), "mutexUnlock")
		defer task.End()
		trace.Log(ctx, "mutexID", fmt.Sprintf("%d", muID))
		trace.Log(ctx, "goroutineID", fmt.Sprintf("%d", gID))

		// Record using our instrumentation
		MutexUnlock(int(muID), gID)
	}
}

// getGoroutineID returns the runtime goroutine ID of the current goroutine
// This is a hack as runtime.GoID() is not exposed in the public API
func getGoroutineID() int64 {
	// Extract the goroutine ID from the stack trace
	buf := make([]byte, 64)
	n := runtime.Stack(buf, false)
	stack := string(buf[:n])

	// Parse the first line which contains the goroutine ID
	idStr := strings.TrimPrefix(strings.Fields(stack)[1], "goroutine")
	idStr = strings.TrimSpace(idStr)
	id, _ := strconv.ParseInt(idStr, 10, 64)
	return id
}

// getGoroutineIDOrAssign returns our internal goroutine ID for current goroutine
func getGoroutineIDOrAssign() int32 {
	runtimeGID := getGoroutineID()

	// Check if we already have this goroutine ID mapped
	if val, ok := traceInt.goroutineMap.Load(runtimeGID); ok {
		// Make sure we get an int32, not interface{} cast to int32
		switch v := val.(type) {
		case int32:
			return v
		case int:
			return int32(v)
		case int64:
			return int32(v)
		default:
			// If it's some other type, assign a new ID to be safe
			// This shouldn't normally happen
			fmt.Printf("Warning: Unexpected type in goroutine map: %T\n", val)
		}
	}

	// Assign a new ID
	ourGID := traceInt.nextGoroutineID
	traceInt.nextGoroutineID++
	traceInt.goroutineMap.Store(runtimeGID, ourGID)

	// Record the new goroutine
	GoroutineCreate(int(ourGID))

	return ourGID
}
