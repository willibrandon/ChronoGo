package recorder

import "time"

// EventType represents the type of an event
type EventType int

const (
	// FuncEntry indicates entering a function
	FuncEntry EventType = iota
	// FuncExit indicates exiting a function
	FuncExit
	// VarAssignment indicates a variable assignment
	VarAssignment
	// GoroutineSwitch indicates a switch between goroutines
	GoroutineSwitch
	// StatementExecution indicates execution of a specific statement
	StatementExecution
	// ChannelOperation indicates a channel operation (send/receive/close)
	ChannelOperation
	// SyncOperation indicates a synchronization primitive operation (mutex lock/unlock)
	SyncOperation
	// SnapshotEvent indicates a state snapshot was created
	SnapshotEvent
	// ... add more as needed
)

// Event represents a recorded event in the program execution
type Event struct {
	ID        int64     // Unique ID of the event
	Timestamp time.Time // Time the event occurred
	Type      EventType // Type of the event
	Details   string    // Human-readable details
	File      string    // Source file where the event occurred
	Line      int       // Line number where the event occurred
	FuncName  string    // Function name where the event occurred
}

// String returns a human-readable representation of the event type
func (et EventType) String() string {
	switch et {
	case FuncEntry:
		return "FunctionEntry"
	case FuncExit:
		return "FunctionExit"
	case VarAssignment:
		return "VariableAssignment"
	case GoroutineSwitch:
		return "GoroutineSwitch"
	case StatementExecution:
		return "StatementExecution"
	case ChannelOperation:
		return "ChannelOperation"
	case SyncOperation:
		return "SyncOperation"
	case SnapshotEvent:
		return "SnapshotEvent"
	default:
		return "Unknown"
	}
}

// Configuration options for ChronoGo
var (
	// SnapshotInterval determines how often snapshots are created (every N events)
	// 0 means no automatic snapshots
	SnapshotInterval = 1000
)

// CurrentTime returns the current time, allowing for testing overrides
func CurrentTime() time.Time {
	return time.Now()
}
