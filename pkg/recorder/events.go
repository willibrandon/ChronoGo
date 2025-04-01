package recorder

import "time"

type EventType int

const (
	FuncEntry EventType = iota
	FuncExit
	VarAssignment
	GoroutineSwitch
	// ... add more as needed
)

type Event struct {
	ID        int64
	Timestamp time.Time
	Type      EventType
	Details   string // e.g., function name, variable info, etc.
}
