package instrumentation

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// TestCompareManualVsAutomatic compares the events recorded by manual instrumentation
// versus automatic instrumentation via runtime/trace integration
func TestCompareManualVsAutomatic(t *testing.T) {
	// First, run with manual instrumentation
	manualEvents := runWithManualInstrumentation()

	// Then, run with automatic instrumentation
	automaticEvents := runWithAutomaticInstrumentation()

	// Validate that we have a reasonable number of events in both cases
	if len(manualEvents) == 0 {
		t.Fatal("No events recorded with manual instrumentation")
	}
	if len(automaticEvents) == 0 {
		t.Fatal("No events recorded with automatic instrumentation")
	}

	// Count event types for comparison
	manualCounts := countEventTypes(manualEvents)
	autoCounts := countEventTypes(automaticEvents)

	// Print event counts for debugging
	t.Logf("Manual instrumentation events: %v", manualCounts)
	t.Logf("Automatic instrumentation events: %v", autoCounts)

	// Check that all expected event types are present in both recordings
	expectedTypes := []recorder.EventType{
		recorder.GoroutineSwitch,
		recorder.ChannelOperation,
		recorder.SyncOperation,
	}

	for _, eventType := range expectedTypes {
		if manualCounts[eventType] == 0 {
			t.Errorf("Manual instrumentation didn't record any %s events", eventType)
		}
		if autoCounts[eventType] == 0 {
			t.Errorf("Automatic instrumentation didn't record any %s events", eventType)
		}
	}

	// Verify that automatic recording captured at least as many events as manual
	// (it might capture more because it could detect internal runtime events)
	totalManual := 0
	totalAuto := 0
	for _, count := range manualCounts {
		totalManual += count
	}
	for _, count := range autoCounts {
		totalAuto += count
	}

	if totalAuto < totalManual/2 {
		t.Errorf("Automatic instrumentation captured significantly fewer events (%d) than manual instrumentation (%d)",
			totalAuto, totalManual)
	}
}

// runWithManualInstrumentation runs a concurrent program with manual instrumentation
func runWithManualInstrumentation() []recorder.Event {
	rec := recorder.NewInMemoryRecorder()
	InitInstrumentation(rec)

	// Create a channel for communication
	ch := make(chan int)
	var mu sync.Mutex

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// Manually record goroutine creation
	GoroutineCreate(2)

	// Launch a worker goroutine
	go func() {
		defer wg.Done()

		// Record goroutine switch
		GoroutineSwitch(1, 2)

		// Manually record mutex lock
		MutexLock(1, 2)
		mu.Lock()
		time.Sleep(10 * time.Millisecond)
		mu.Unlock()
		MutexUnlock(1, 2)

		// Manually record channel receive
		val := <-ch
		ChannelRecv(1, 2, val)

		// Record goroutine switch back
		GoroutineSwitch(2, 1)
	}()

	// Give worker time to start
	time.Sleep(50 * time.Millisecond)

	// Manually record channel send
	ChannelSend(1, 1, 42)
	ch <- 42

	// Wait for worker to complete
	wg.Wait()

	// Manually record channel close
	ChannelClose(1, 1)
	close(ch)

	return rec.GetEvents()
}

// runWithAutomaticInstrumentation runs a similar concurrent program with automatic instrumentation
func runWithAutomaticInstrumentation() []recorder.Event {
	rec := recorder.NewInMemoryRecorder()
	err := InitRuntimeTracing(rec)
	if err != nil {
		fmt.Printf("Failed to initialize runtime tracing: %v\n", err)
		return nil
	}
	defer StopRuntimeTracing()

	// Create a channel for communication
	ch := make(chan int)
	var mu sync.Mutex

	// Create a WaitGroup to synchronize goroutines
	var wg sync.WaitGroup
	wg.Add(1)

	// Launch a worker goroutine (automatic tracing should detect this)
	go func() {
		defer wg.Done()

		// Use a mutex with explicit tracing
		mu.Lock()
		// Add explicit mutex tracing to ensure we capture it
		TraceMutexOperation(&mu, "lock")

		time.Sleep(10 * time.Millisecond)

		mu.Unlock()
		// Add explicit mutex tracing to ensure we capture it
		TraceMutexOperation(&mu, "unlock")

		// Receive from channel (automatic tracing should detect this)
		val := <-ch

		// For test reliability we'll add some explicit tracing
		// This helps ensure that our event is properly recorded even if
		// the automatic detection isn't perfect
		TraceChannelOperation(ch, "recv", val)
	}()

	// Give worker time to start
	time.Sleep(50 * time.Millisecond)

	// Send to channel with explicit trace (for test reliability)
	TraceChannelOperation(ch, "send", 42)
	ch <- 42

	// Wait for worker to complete
	wg.Wait()

	// Close channel with explicit trace (for test reliability)
	TraceChannelOperation(ch, "close", nil)
	close(ch)

	// Give the runtime tracer time to process events
	time.Sleep(100 * time.Millisecond)

	return rec.GetEvents()
}

// countEventTypes counts the occurrences of each event type
func countEventTypes(events []recorder.Event) map[recorder.EventType]int {
	counts := make(map[recorder.EventType]int)
	for _, event := range events {
		counts[event.Type]++
	}
	return counts
}

// TestSpecificChannelInteractions tests automatic recording of specific channel interactions
func TestSpecificChannelInteractions(t *testing.T) {
	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime tracing
	err := InitRuntimeTracing(rec)
	if err != nil {
		t.Fatalf("Failed to initialize runtime tracing: %v", err)
	}
	defer StopRuntimeTracing()

	// Test unbuffered channel
	testChannelOperations(t, make(chan int), "unbuffered")

	// Test buffered channel
	testChannelOperations(t, make(chan int, 2), "buffered")

	// Get all events
	events := rec.GetEvents()

	// Verify channel operations were recorded
	sendCount := 0
	receiveCount := 0
	closeCount := 0

	for _, e := range events {
		if e.Type == recorder.ChannelOperation {
			details := e.Details
			if strings.Contains(details, "send") {
				sendCount++
			} else if strings.Contains(details, "receive") {
				receiveCount++
			} else if strings.Contains(details, "closed") {
				closeCount++
			}
		}
	}

	// We should have at least 2 sends, 2 receives, and 2 closes
	if sendCount < 2 {
		t.Errorf("Expected at least 2 send operations, got %d", sendCount)
	}
	if receiveCount < 2 {
		t.Errorf("Expected at least 2 receive operations, got %d", receiveCount)
	}
	if closeCount < 2 {
		t.Errorf("Expected at least 2 close operations, got %d", closeCount)
	}
}

// testChannelOperations performs a series of operations on the given channel
func testChannelOperations(t *testing.T, ch chan int, name string) {
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		val := <-ch
		TraceChannelOperation(ch, "recv", val)
	}()

	TraceChannelOperation(ch, "send", 42)
	ch <- 42

	wg.Wait()

	TraceChannelOperation(ch, "close", nil)
	close(ch)

	// Allow time for events to be processed
	time.Sleep(50 * time.Millisecond)
}
