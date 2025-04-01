package tests

import (
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

func TestBasicReplayer(t *testing.T) {
	// Create some test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering testFunc",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(100 * time.Millisecond),
			Type:      recorder.FuncExit,
			Details:   "Exiting testFunc",
		},
	}

	// Create replayer and load events
	replayer := replay.NewBasicReplayer()
	err := replayer.LoadEvents(events)
	if err != nil {
		t.Errorf("Failed to load events: %v", err)
	}

	// Verify replay
	err = replayer.ReplayForward()
	if err != nil {
		t.Errorf("Failed to replay events: %v", err)
	}

	// Get the loaded events from replayer
	replayedEvents := replayer.Events()
	if len(replayedEvents) != 2 {
		t.Errorf("Expected 2 events, got %d", len(replayedEvents))
	}

	// Verify events are in correct order
	if replayedEvents[0].ID != 1 || replayedEvents[1].ID != 2 {
		t.Error("Events are not in correct order")
	}

	// Verify timestamps are in order
	if !replayedEvents[0].Timestamp.Before(replayedEvents[1].Timestamp) {
		t.Error("Event timestamps are not in correct order")
	}
}
