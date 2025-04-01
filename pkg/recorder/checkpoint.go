package recorder

import (
	"fmt"
	"time"
)

// Checkpoint represents a point in time during program execution
// that we can restore to and replay forward from
type Checkpoint struct {
	ID        int64
	Snapshot  Snapshot
	EventIdx  int
	Timestamp time.Time
}

// NewCheckpoint creates a new checkpoint with the given snapshot and event index
func NewCheckpoint(snapshot Snapshot, eventIdx int) *Checkpoint {
	return &Checkpoint{
		ID:        time.Now().UnixNano(),
		Snapshot:  snapshot,
		EventIdx:  eventIdx,
		Timestamp: time.Now(),
	}
}

// String returns a human-readable representation of the checkpoint
func (c *Checkpoint) String() string {
	return fmt.Sprintf("Checkpoint{ID: %d, EventIdx: %d, Time: %s}",
		c.ID, c.EventIdx, c.Timestamp.Format(time.RFC3339))
}
