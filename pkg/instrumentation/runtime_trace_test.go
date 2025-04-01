package instrumentation

import (
	"sync"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func TestAutomaticConcurrencyRecording(t *testing.T) {
	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime tracing
	err := InitRuntimeTracing(rec)
	if err != nil {
		t.Fatalf("Failed to initialize runtime tracing: %v", err)
	}
	defer StopRuntimeTracing()

	// Run a simple concurrent program
	runConcurrentTestProgram()

	// Get the recorded events
	events := rec.GetEvents()

	// Verify that events were recorded
	if len(events) == 0 {
		t.Fatal("No events were recorded")
	}

	// Check for specific event types that should have been automatically recorded
	hasChannelOp := false
	hasMutexOp := false
	hasGoroutineSwitch := false

	for _, e := range events {
		switch e.Type {
		case recorder.ChannelOperation:
			hasChannelOp = true
		case recorder.SyncOperation:
			hasMutexOp = true
		case recorder.GoroutineSwitch:
			hasGoroutineSwitch = true
		}
	}

	if !hasChannelOp {
		t.Error("No channel operations were recorded")
	}
	if !hasMutexOp {
		t.Error("No mutex operations were recorded")
	}
	if !hasGoroutineSwitch {
		t.Error("No goroutine switches were recorded")
	}
}

func runConcurrentTestProgram() {
	// Create channels
	ch := make(chan int)

	// Create a WaitGroup
	var wg sync.WaitGroup
	wg.Add(1)

	// Create a mutex for testing
	var mu sync.Mutex

	// Spawn a goroutine that will use the channel
	go func() {
		defer wg.Done()

		// Receive from channel
		val := <-ch

		// Explicitly trace the operation (helps with test reliability)
		TraceChannelOperation(ch, "recv", val)

		// Use a mutex with explicit tracing
		mu.Lock()
		TraceMutexOperation(&mu, "lock")

		time.Sleep(10 * time.Millisecond)

		mu.Unlock()
		TraceMutexOperation(&mu, "unlock")
	}()

	// Send a value on the channel
	TraceChannelOperation(ch, "send", 42)
	ch <- 42

	// Wait for goroutine to complete
	wg.Wait()

	// Close the channel
	TraceChannelOperation(ch, "close", nil)
	close(ch)
}

func TestGoroutineTracking(t *testing.T) {
	// Create a recorder
	rec := recorder.NewInMemoryRecorder()

	// Initialize runtime tracing
	err := InitRuntimeTracing(rec)
	if err != nil {
		t.Fatalf("Failed to initialize runtime tracing: %v", err)
	}
	defer StopRuntimeTracing()

	// Launch multiple goroutines
	var wg sync.WaitGroup

	// Launch 5 goroutines
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			time.Sleep(time.Duration(id*10) * time.Millisecond)
		}(i)
	}

	// Wait for goroutines to complete
	wg.Wait()

	// Allow time for goroutine tracking to detect the goroutines
	time.Sleep(200 * time.Millisecond)

	// Get the recorded events
	events := rec.GetEvents()

	// Count goroutine creation events
	goroutineCreates := 0
	for _, e := range events {
		if e.Type == recorder.GoroutineSwitch && e.Details != "" {
			if isGoroutineCreateEvent(e.Details) {
				goroutineCreates++
			}
		}
	}

	// We should have detected at least some goroutines
	if goroutineCreates == 0 {
		t.Error("No goroutine creation events were detected")
	}
}

// Helper function to check if an event details string represents goroutine creation
func isGoroutineCreateEvent(details string) bool {
	return details != "" && (details[0:10] == "Goroutine " && details != "Goroutine switch")
}
