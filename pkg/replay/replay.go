package replay

import (
	"fmt"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// Replayer defines the interface for replaying recorded events
type Replayer interface {
	LoadEvents(events []recorder.Event) error
	ReplayForward() error
	Events() []recorder.Event
}

// BasicReplayer implements the Replayer interface
type BasicReplayer struct {
	events []recorder.Event
}

// NewBasicReplayer creates a new instance of BasicReplayer
func NewBasicReplayer() Replayer {
	return &BasicReplayer{
		events: make([]recorder.Event, 0),
	}
}

// LoadEvents loads the provided events into the replayer
func (r *BasicReplayer) LoadEvents(events []recorder.Event) error {
	r.events = events
	return nil
}

// ReplayForward replays the events in forward order
func (r *BasicReplayer) ReplayForward() error {
	fmt.Println("Starting forward replay...")
	for _, event := range r.events {
		fmt.Printf("[%s] Event %d: %s - %s\n",
			event.Timestamp.Format(time.RFC3339),
			event.ID,
			event.Type,
			event.Details)
		time.Sleep(100 * time.Millisecond) // Simulate time between events
	}
	fmt.Println("Replay complete")
	return nil
}

// Events returns the loaded events
func (r *BasicReplayer) Events() []recorder.Event {
	return r.events
}
