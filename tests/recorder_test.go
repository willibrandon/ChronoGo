package tests

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func TestFileRecorder(t *testing.T) {
	// Create a temporary file for testing
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test_events.log")

	// Create a new FileRecorder
	r, err := recorder.NewFileRecorder(logPath)
	if err != nil {
		t.Fatalf("Failed to create FileRecorder: %v", err)
	}
	defer r.Close()

	// Record a test event
	testEvent := recorder.Event{
		ID:        time.Now().UnixNano(),
		Timestamp: time.Now(),
		Type:      recorder.FuncEntry,
		Details:   "Test Function Entry",
	}

	if err := r.RecordEvent(testEvent); err != nil {
		t.Errorf("Failed to record event: %v", err)
	}

	// Read back the events
	events := r.GetEvents()
	if len(events) != 1 {
		t.Errorf("Expected 1 event, got %d", len(events))
	}

	// Verify event contents
	if events[0].Details != testEvent.Details {
		t.Errorf("Event details mismatch. Got %s, want %s", events[0].Details, testEvent.Details)
	}

	// Test Clear functionality
	r.Clear()
	events = r.GetEvents()
	if len(events) != 0 {
		t.Errorf("Expected 0 events after Clear(), got %d", len(events))
	}
}

func TestSnapshot(t *testing.T) {
	// Create a snapshot with a known ID
	testID := time.Now().UnixNano()
	snapshot := recorder.CreateSnapshot(testID)

	// Verify snapshot
	if snapshot.ID != testID {
		t.Errorf("Snapshot ID mismatch. Got %d, want %d", snapshot.ID, testID)
	}

	// Verify mock state is present
	if len(snapshot.MemDump) == 0 {
		t.Error("Expected non-empty MemDump in snapshot")
	}

	expectedState := []byte("mock state")
	if string(snapshot.MemDump) != string(expectedState) {
		t.Errorf("MemDump content mismatch. Got %s, want %s",
			string(snapshot.MemDump), string(expectedState))
	}
}
