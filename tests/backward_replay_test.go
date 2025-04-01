package tests

import (
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

func TestBackwardExecution(t *testing.T) {
	// Create test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering main",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(100 * time.Millisecond),
			Type:      recorder.FuncEntry,
			Details:   "Entering testFunc",
		},
		{
			ID:        3,
			Timestamp: time.Now().Add(200 * time.Millisecond),
			Type:      recorder.FuncExit,
			Details:   "Exiting testFunc",
		},
		{
			ID:        4,
			Timestamp: time.Now().Add(300 * time.Millisecond),
			Type:      recorder.FuncExit,
			Details:   "Exiting main",
		},
	}

	// Create replayer and load events
	replayer := replay.NewBasicReplayer()
	err := replayer.LoadEvents(events)
	if err != nil {
		t.Fatalf("Failed to load events: %v", err)
	}

	// First replay forward to the end
	err = replayer.ReplayForward()
	if err != nil {
		t.Fatalf("Failed to replay forward: %v", err)
	}

	// Verify current index is at the end
	if replayer.CurrentIndex() != len(events)-1 {
		t.Errorf("Expected current index to be %d, got %d", len(events)-1, replayer.CurrentIndex())
	}

	// Step backward one event
	newIdx, err := replayer.StepBackward(replayer.CurrentIndex())
	if err != nil {
		t.Fatalf("Failed to step backward: %v", err)
	}

	// Verify we moved back one event
	if newIdx != len(events)-2 {
		t.Errorf("Expected to step back to index %d, got %d", len(events)-2, newIdx)
	}

	// Try to replay to a specific index
	targetIdx := 1 // Should replay to "Entering testFunc"
	err = replayer.ReplayToEventIndex(targetIdx)
	if err != nil {
		t.Fatalf("Failed to replay to index %d: %v", targetIdx, err)
	}

	// Verify current index is at the target
	if replayer.CurrentIndex() != targetIdx {
		t.Errorf("Expected current index to be %d after replay, got %d", targetIdx, replayer.CurrentIndex())
	}

	// Try to step backward from the beginning (should fail gracefully)
	_, err = replayer.StepBackward(0)
	if err == nil {
		t.Error("Expected error when stepping backward from beginning, got nil")
	}
}
