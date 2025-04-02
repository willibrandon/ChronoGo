package replay

import (
	"fmt"
	"strings"
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

// GoroutineState tracks the state of a goroutine
type GoroutineState struct {
	ID      int
	Running bool
}

// ChannelState tracks the state of a channel
type ChannelState struct {
	ID       int
	Messages []interface{}
	Closed   bool
}

// BasicReplayer implements the Replayer interface
type BasicReplayer struct {
	events          []recorder.Event
	currentIdx      int
	goroutines      map[int]*GoroutineState // Track goroutine states
	channels        map[int]*ChannelState   // Track channel states
	activeGoroutine int                     // Currently active goroutine
}

// NewBasicReplayer creates a new BasicReplayer
func NewBasicReplayer() *BasicReplayer {
	return &BasicReplayer{
		events:          []recorder.Event{},
		currentIdx:      -1,
		goroutines:      make(map[int]*GoroutineState),
		channels:        make(map[int]*ChannelState),
		activeGoroutine: 1, // Start with main goroutine (ID 1)
	}
}

// LoadEvents loads the given events into the replayer
func (r *BasicReplayer) LoadEvents(events []recorder.Event) error {
	r.events = events
	r.currentIdx = -1

	// Initialize concurrency tracking
	r.goroutines = make(map[int]*GoroutineState)
	r.channels = make(map[int]*ChannelState)
	r.activeGoroutine = 1 // Reset to main goroutine

	// Initialize the main goroutine
	r.goroutines[1] = &GoroutineState{ID: 1, Running: true}

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

		// Process concurrency events to update goroutine and channel states
		r.processGoroutineAndChannelEvents(event)

		// Check for variable changes in statements that might trigger a watchpoint
		if event.Type == recorder.StatementExecution {
			// Look for variable assignments in the details
			details := event.Details
			if strings.Contains(details, " = ") {
				// This could be a variable assignment that would trigger a watchpoint
				fmt.Printf("DEBUG: Potential variable change detected: %s\n", details)
			}
		}

		// Check if this event hits a breakpoint BEFORE reporting
		if haveBreakpointCheck && breakpointCheck(event) {
			fmt.Printf("Breakpoint hit at event %d\n", i)
			r.currentIdx = i
			return nil
		}

		// Print event details with goroutine info for concurrency events
		if event.Type == recorder.GoroutineSwitch ||
			event.Type == recorder.ChannelOperation ||
			event.Type == recorder.SyncOperation {
			fmt.Printf("[%s] Event %d: %s (Goroutine %d)\n",
				event.Timestamp.Format(time.RFC3339),
				event.ID,
				event.Details,
				r.activeGoroutine)
		} else {
			fmt.Printf("[%s] Event %d: %s\n",
				event.Timestamp.Format(time.RFC3339),
				event.ID,
				event.Details)
		}

		r.currentIdx = i
		time.Sleep(50 * time.Millisecond) // Simulate time between events
	}

	fmt.Println("Replay complete")
	return nil
}

// processGoroutineAndChannelEvents updates the internal state based on concurrency events
func (r *BasicReplayer) processGoroutineAndChannelEvents(event recorder.Event) {
	switch event.Type {
	case recorder.GoroutineSwitch:
		// Handle goroutine creation or switching
		if strings.Contains(event.Details, "created") {
			// Extract goroutine ID from the details
			var gID int
			_, err := fmt.Sscanf(event.Details, "Goroutine %d created", &gID)
			if err != nil {
				// If we can't parse the goroutine ID, use a default
				gID = 0
				fmt.Printf("Warning: Could not parse goroutine ID from %s: %v\n", event.Details, err)
			}
			r.goroutines[gID] = &GoroutineState{ID: gID, Running: true}
		} else if strings.Contains(event.Details, "switch from") {
			// Extract from and to goroutine IDs
			var fromID, toID int
			_, err := fmt.Sscanf(event.Details, "Goroutine switch from %d to %d", &fromID, &toID)
			if err != nil {
				// If we can't parse the goroutine IDs, use defaults
				fmt.Printf("Warning: Could not parse goroutine switch IDs from %s: %v\n", event.Details, err)
				return
			}
			if g, exists := r.goroutines[fromID]; exists {
				g.Running = false
			}
			if g, exists := r.goroutines[toID]; exists {
				g.Running = true
			} else {
				// Create it if it doesn't exist
				r.goroutines[toID] = &GoroutineState{ID: toID, Running: true}
			}
			r.activeGoroutine = toID
		}

	case recorder.ChannelOperation:
		// Handle channel operations (send, receive, close)
		if strings.Contains(event.Details, "send by") {
			// Extract channel ID, goroutine ID, and value
			var chID, gID int
			_, err := fmt.Sscanf(event.Details, "Channel %d: send by goroutine %d", &chID, &gID)
			if err != nil {
				// If we can't parse the channel and goroutine IDs, use defaults
				fmt.Printf("Warning: Could not parse channel send IDs from %s: %v\n", event.Details, err)
				return
			}

			// Ensure the channel exists in our map
			if _, exists := r.channels[chID]; !exists {
				r.channels[chID] = &ChannelState{ID: chID, Messages: []interface{}{}, Closed: false}
			}

		} else if strings.Contains(event.Details, "receive by") {
			// Extract channel ID and goroutine ID
			var chID, gID int
			_, err := fmt.Sscanf(event.Details, "Channel %d: receive by goroutine %d", &chID, &gID)
			if err != nil {
				// If we can't parse the channel and goroutine IDs, use defaults
				fmt.Printf("Warning: Could not parse channel receive IDs from %s: %v\n", event.Details, err)
				return
			}

			// Ensure the channel exists
			if _, exists := r.channels[chID]; !exists {
				r.channels[chID] = &ChannelState{ID: chID, Messages: []interface{}{}, Closed: false}
			}

		} else if strings.Contains(event.Details, "closed by") {
			// Extract channel ID and goroutine ID
			var chID, gID int
			_, err := fmt.Sscanf(event.Details, "Channel %d: closed by goroutine %d", &chID, &gID)
			if err != nil {
				// If we can't parse the channel and goroutine IDs, use defaults
				fmt.Printf("Warning: Could not parse channel close IDs from %s: %v\n", event.Details, err)
				return
			}

			// Mark the channel as closed
			if ch, exists := r.channels[chID]; exists {
				ch.Closed = true
			}
		}
	}
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
