package replay

import (
	"fmt"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// Replayer interface defines methods for replaying recorded events
type Replayer interface {
	// LoadEvents loads recorded events into the replayer
	LoadEvents([]recorder.Event) error

	// ReplayForward replays all events from the current position
	ReplayForward() error

	// ReplayUntilBreakpoint replays events until a breakpoint is hit
	ReplayUntilBreakpoint(breakpointCheck func(event recorder.Event) bool) error

	// ReplayToEventIndex replays events up to the specified index
	ReplayToEventIndex(idx int) error

	// StepBackward steps backward from the current index
	// returns the new index after stepping back
	StepBackward(currentIdx int) (int, error)

	// CurrentIndex returns the current event index
	CurrentIndex() int

	// Events returns all loaded events
	Events() []recorder.Event
}

// BasicReplayer implements the Replayer interface
type BasicReplayer struct {
	events     []recorder.Event
	currentIdx int
}

// NewBasicReplayer creates a new BasicReplayer
func NewBasicReplayer() *BasicReplayer {
	return &BasicReplayer{
		events:     []recorder.Event{},
		currentIdx: -1,
	}
}

// LoadEvents loads the given events into the replayer
func (r *BasicReplayer) LoadEvents(events []recorder.Event) error {
	r.events = events
	r.currentIdx = -1
	return nil
}

// ReplayForward replays all events from current position to the end
func (r *BasicReplayer) ReplayForward() error {
	return r.ReplayUntilBreakpoint(nil)
}

// ReplayUntilBreakpoint replays events until a breakpoint is hit
// If breakpointCheck is nil, replay all events
func (r *BasicReplayer) ReplayUntilBreakpoint(breakpointCheck func(event recorder.Event) bool) error {
	if len(r.events) == 0 {
		return nil
	}

	startIdx := r.currentIdx + 1
	if startIdx < 0 {
		startIdx = 0
	}

	// Check if we have any breakpoints to check
	haveBreakpointCheck := breakpointCheck != nil

	// Replay events until end or breakpoint hit
	for i := startIdx; i < len(r.events); i++ {
		event := r.events[i]

		// Check if this event hits a breakpoint BEFORE reporting
		if haveBreakpointCheck && breakpointCheck(event) {
			fmt.Printf("Breakpoint hit at event %d\n", i)
			r.currentIdx = i
			return nil
		}

		// Print event details
		fmt.Printf("[%s] Event %d: %s - %s\n",
			event.Timestamp.Format(time.RFC3339),
			event.ID,
			event.Type,
			event.Details)

		r.currentIdx = i
		time.Sleep(50 * time.Millisecond) // Simulate time between events
	}

	fmt.Println("Replay complete")
	return nil
}

// ReplayToEventIndex replays events up to the specified index
func (r *BasicReplayer) ReplayToEventIndex(idx int) error {
	if idx < 0 || idx >= len(r.events) {
		return nil
	}

	r.currentIdx = idx
	return nil
}

// StepBackward moves one step backward in the event log
func (r *BasicReplayer) StepBackward(currentIdx int) (int, error) {
	if currentIdx <= 0 {
		return 0, fmt.Errorf("already at the beginning")
	}

	newIdx := currentIdx - 1
	r.currentIdx = newIdx
	return newIdx, nil
}

// CurrentIndex returns the current event index
func (r *BasicReplayer) CurrentIndex() int {
	return r.currentIdx
}

// Events returns all loaded events
func (r *BasicReplayer) Events() []recorder.Event {
	return r.events
}
