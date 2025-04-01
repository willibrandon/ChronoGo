package recorder

import (
	"fmt"
)

// Checkpoint represents a point in time in the execution of a program
// It combines a snapshot of program state with the event index
type Checkpoint struct {
	Snapshot Snapshot
	EventIdx int
}

// NewCheckpoint creates a new checkpoint
func NewCheckpoint(snapshot Snapshot, eventIdx int) *Checkpoint {
	return &Checkpoint{
		Snapshot: snapshot,
		EventIdx: eventIdx,
	}
}

// String returns a human-readable representation of the checkpoint
func (c *Checkpoint) String() string {
	return fmt.Sprintf("Checkpoint{EventIdx: %d}", c.EventIdx)
}
