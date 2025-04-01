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
	instrumentation.FuncEntry("TestFunction")
	time.Sleep(10 * time.Millisecond) // Small delay to ensure different timestamps
	instrumentation.FuncExit("TestFunction")

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

	// Verify function names in event details
	if events[0].Details != "Entering TestFunction" {
		t.Errorf("Unexpected entry details: %s", events[0].Details)
	}
	if events[1].Details != "Exiting TestFunction" {
		t.Errorf("Unexpected exit details: %s", events[1].Details)
	}

	// Verify timestamps are in order
	if !events[0].Timestamp.Before(events[1].Timestamp) {
		t.Error("Entry timestamp should be before exit timestamp")
	}
}
