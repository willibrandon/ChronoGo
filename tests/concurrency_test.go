package tests

import (
	"strings"
	"testing"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// TestGoroutineTracking tests that goroutine creation and switching events are properly recorded
func TestGoroutineTracking(t *testing.T) {
	// Create an in-memory recorder for testing
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	// Simulate goroutine operations
	// In practice, these would be called by the runtime hooks
	instrumentation.GoroutineCreate(2)
	instrumentation.GoroutineSwitch(1, 2) // Switch from main goroutine to goroutine 2
	instrumentation.GoroutineSwitch(2, 1) // Switch back to main goroutine

	// Verify events were recorded
	events := rec.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Check types and details
	if events[0].Type != recorder.GoroutineSwitch ||
		!strings.Contains(events[0].Details, "created") {
		t.Errorf("First event should be goroutine creation, got %s", events[0].Details)
	}

	if events[1].Type != recorder.GoroutineSwitch ||
		!strings.Contains(events[1].Details, "switch from") {
		t.Errorf("Second event should be goroutine switch, got %s", events[1].Details)
	}
}

// TestChannelOperations tests that channel operations are properly recorded
func TestChannelOperations(t *testing.T) {
	// Create an in-memory recorder for testing
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	// Simulate channel operations
	instrumentation.ChannelSend(1, 1, 42) // Channel 1, goroutine 1, value 42
	instrumentation.ChannelRecv(1, 2, 42) // Channel 1, goroutine 2, value 42
	instrumentation.ChannelClose(1, 1)    // Channel 1, closed by goroutine 1

	// Verify events were recorded
	events := rec.GetEvents()
	if len(events) != 3 {
		t.Fatalf("Expected 3 events, got %d", len(events))
	}

	// Check types and details
	if events[0].Type != recorder.ChannelOperation ||
		!strings.Contains(events[0].Details, "send by") {
		t.Errorf("First event should be channel send, got %s", events[0].Details)
	}

	if events[1].Type != recorder.ChannelOperation ||
		!strings.Contains(events[1].Details, "receive by") {
		t.Errorf("Second event should be channel receive, got %s", events[1].Details)
	}

	if events[2].Type != recorder.ChannelOperation ||
		!strings.Contains(events[2].Details, "closed by") {
		t.Errorf("Third event should be channel close, got %s", events[2].Details)
	}
}

// TestMutexOperations tests that mutex operations are properly recorded
func TestMutexOperations(t *testing.T) {
	// Create an in-memory recorder for testing
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	// Simulate mutex operations
	instrumentation.MutexLock(1, 1)   // Mutex 1, goroutine 1
	instrumentation.MutexUnlock(1, 1) // Mutex 1, goroutine 1

	// Verify events were recorded
	events := rec.GetEvents()
	if len(events) != 2 {
		t.Fatalf("Expected 2 events, got %d", len(events))
	}

	// Check types and details
	if events[0].Type != recorder.SyncOperation ||
		!strings.Contains(events[0].Details, "locked by") {
		t.Errorf("First event should be mutex lock, got %s", events[0].Details)
	}

	if events[1].Type != recorder.SyncOperation ||
		!strings.Contains(events[1].Details, "unlocked by") {
		t.Errorf("Second event should be mutex unlock, got %s", events[1].Details)
	}
}

// TestConcurrencyReplay tests the replay of concurrency events
func TestConcurrencyReplay(t *testing.T) {
	// Create an in-memory recorder for testing
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	// Simulate a series of concurrency events
	instrumentation.GoroutineCreate(2)
	instrumentation.GoroutineSwitch(1, 2)
	instrumentation.ChannelSend(1, 2, 42)
	instrumentation.GoroutineSwitch(2, 1)
	instrumentation.ChannelRecv(1, 1, 42)
	instrumentation.MutexLock(1, 1)
	instrumentation.MutexUnlock(1, 1)
	instrumentation.ChannelClose(1, 1)

	// Create a replayer and load the events
	replayer := replay.NewBasicReplayer()
	err := replayer.LoadEvents(rec.GetEvents())
	if err != nil {
		t.Fatalf("Failed to load events: %v", err)
	}

	// Verify events can be processed without errors
	err = replayer.ReplayForward()
	if err != nil {
		t.Fatalf("Error replaying events: %v", err)
	}

	// Verify the current index is at the end
	if replayer.CurrentIndex() != len(rec.GetEvents())-1 {
		t.Errorf("Replayer did not reach the end of events, currentIdx: %d, expected: %d",
			replayer.CurrentIndex(), len(rec.GetEvents())-1)
	}
}
