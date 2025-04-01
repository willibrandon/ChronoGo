package tests

import (
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func TestInMemoryRecorder(t *testing.T) {
	// Create a new recorder
	r := recorder.NewInMemoryRecorder()

	// Initialize instrumentation with our recorder
	instrumentation.InitInstrumentation(r)

	// Record some test events
	instrumentation.FuncEntry("TestFunction", "tests/instrumentation_test.go", 19)
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	instrumentation.FuncExit("TestFunction", "tests/instrumentation_test.go", 21)

	// Get recorded events
	events := r.GetEvents()

	// Verify we got exactly 2 events
	if len(events) != 2 {
		t.Errorf("Expected 2 events, got %d", len(events))
	}

	// Verify first event is FuncEntry
	if events[0].Type != recorder.FuncEntry {
		t.Errorf("First event should be FuncEntry, got %v", events[0].Type)
	}

	// Verify second event is FuncExit
	if events[1].Type != recorder.FuncExit {
		t.Errorf("Second event should be FuncExit, got %v", events[1].Type)
	}

	// Verify function names and file information in event details
	expectedEntryDetails := "Entering TestFunction at tests/instrumentation_test.go:19"
	if events[0].Details != expectedEntryDetails {
		t.Errorf("Unexpected entry details: %s, expected: %s", events[0].Details, expectedEntryDetails)
	}

	expectedExitDetails := "Exiting TestFunction at tests/instrumentation_test.go:21"
	if events[1].Details != expectedExitDetails {
		t.Errorf("Unexpected exit details: %s, expected: %s", events[1].Details, expectedExitDetails)
	}

	// Verify timestamps are in order
	if !events[0].Timestamp.Before(events[1].Timestamp) {
		t.Error("Entry timestamp should be before exit timestamp")
	}

	// Verify file and line information is preserved
	if events[0].File != "tests/instrumentation_test.go" || events[0].Line != 19 {
		t.Errorf("Function entry file/line incorrect: got %s:%d, expected tests/instrumentation_test.go:19",
			events[0].File, events[0].Line)
	}

	if events[1].File != "tests/instrumentation_test.go" || events[1].Line != 21 {
		t.Errorf("Function exit file/line incorrect: got %s:%d, expected tests/instrumentation_test.go:21",
			events[1].File, events[1].Line)
	}
}
