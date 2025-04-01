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
	ReplayToEventIndex(idx int) error
	StepBackward(currentIdx int) (int, error)
	Events() []recorder.Event
	CurrentIndex() int
}

// BasicReplayer implements the Replayer interface
type BasicReplayer struct {
	events       []recorder.Event
	checkpoints  []*recorder.Checkpoint
	currentIndex int
}

// NewBasicReplayer creates a new instance of BasicReplayer
func NewBasicReplayer() Replayer {
	return &BasicReplayer{
		events:       make([]recorder.Event, 0),
		checkpoints:  make([]*recorder.Checkpoint, 0),
		currentIndex: -1,
	}
}

// LoadEvents loads the provided events into the replayer
func (r *BasicReplayer) LoadEvents(events []recorder.Event) error {
	r.events = events
	r.currentIndex = -1

	// Create initial checkpoint
	if len(events) > 0 {
		snapshot := recorder.Snapshot{
			ID: time.Now().UnixNano(),
		}
		checkpoint := recorder.NewCheckpoint(snapshot, 0)
		r.checkpoints = append(r.checkpoints, checkpoint)
	}

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
		r.currentIndex++
	}
	fmt.Println("Replay complete")
	return nil
}

// ReplayToEventIndex replays events up to the specified index
func (r *BasicReplayer) ReplayToEventIndex(idx int) error {
	if idx < 0 || idx >= len(r.events) {
		return fmt.Errorf("invalid event index: %d", idx)
	}

	// Find the most recent checkpoint before idx
	checkpoint := r.findNearestCheckpoint(idx)
	if checkpoint == nil {
		return fmt.Errorf("no checkpoint found before index %d", idx)
	}

	fmt.Printf("Restoring from checkpoint at event %d\n", checkpoint.EventIdx)

	// Replay from checkpoint to target index
	fmt.Printf("Replaying from event %d to %d\n", checkpoint.EventIdx, idx)
	for i := checkpoint.EventIdx; i <= idx; i++ {
		event := r.events[i]
		fmt.Printf("[%s] Event %d: %s - %s\n",
			event.Timestamp.Format(time.RFC3339),
			event.ID,
			event.Type,
			event.Details)
		time.Sleep(50 * time.Millisecond) // Faster replay for backward steps
	}

	r.currentIndex = idx
	return nil
}

// StepBackward moves execution back one event
func (r *BasicReplayer) StepBackward(currentIdx int) (int, error) {
	if currentIdx <= 0 {
		return 0, fmt.Errorf("already at the beginning")
	}

	targetIdx := currentIdx - 1
	err := r.ReplayToEventIndex(targetIdx)
	if err != nil {
		return currentIdx, fmt.Errorf("failed to step backward: %v", err)
	}

	return targetIdx, nil
}

// findNearestCheckpoint returns the most recent checkpoint before the given index
func (r *BasicReplayer) findNearestCheckpoint(idx int) *recorder.Checkpoint {
	var nearest *recorder.Checkpoint
	for _, cp := range r.checkpoints {
		if cp.EventIdx <= idx && (nearest == nil || cp.EventIdx > nearest.EventIdx) {
			nearest = cp
		}
	}
	return nearest
}

// Events returns the loaded events
func (r *BasicReplayer) Events() []recorder.Event {
	return r.events
}

// CurrentIndex returns the current event index
func (r *BasicReplayer) CurrentIndex() int {
	return r.currentIndex
}
