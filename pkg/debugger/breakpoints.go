package debugger

import (
	"fmt"
	"strconv"
	"strings"
)

// BreakpointType defines the type of breakpoint
type BreakpointType int

const (
	// LocationBreakpoint breaks at a specific file:line
	LocationBreakpoint BreakpointType = iota
	// FunctionBreakpoint breaks at a function entry
	FunctionBreakpoint
	// EventTypeBreakpoint breaks at a specific event type
	EventTypeBreakpoint
	// WatchpointRead breaks when a memory location is read
	WatchpointRead
	// WatchpointWrite breaks when a memory location is written
	WatchpointWrite
	// WatchpointReadWrite breaks when a memory location is read or written
	WatchpointReadWrite
)

// Breakpoint represents a location to stop at during debugging
type Breakpoint struct {
	ID         int
	Type       BreakpointType
	File       string // For LocationBreakpoint
	Line       int    // For LocationBreakpoint
	Function   string // For FunctionBreakpoint
	EventType  string // For EventTypeBreakpoint
	Expression string // For Watchpoint: the expression to watch
	Address    uint64 // For Watchpoint: the memory address to watch (if resolved)
	Enabled    bool
}

// BreakpointManager manages breakpoints for the debugger
type BreakpointManager struct {
	breakpoints []*Breakpoint
	nextID      int
}

// NewBreakpointManager creates a new breakpoint manager
func NewBreakpointManager() *BreakpointManager {
	return &BreakpointManager{
		breakpoints: make([]*Breakpoint, 0),
		nextID:      1,
	}
}

// AddBreakpoint adds a breakpoint at the specified location
func (bm *BreakpointManager) AddBreakpoint(location string) (*Breakpoint, error) {
	bp := &Breakpoint{
		ID:      bm.nextID,
		Enabled: true,
	}
	bm.nextID++

	// Parse location string
	if strings.HasPrefix(location, "func:") {
		// Function breakpoint
		bp.Type = FunctionBreakpoint
		bp.Function = strings.TrimPrefix(location, "func:")
	} else if strings.Contains(location, ":") {
		// Location breakpoint (file:line)
		bp.Type = LocationBreakpoint

		// Find the last colon to handle Windows paths (e.g., C:/path/to/file.go:42)
		lastColonIndex := strings.LastIndex(location, ":")
		if lastColonIndex == -1 {
			return nil, fmt.Errorf("invalid location format: %s", location)
		}

		bp.File = location[:lastColonIndex]
		lineStr := location[lastColonIndex+1:]

		line, err := strconv.Atoi(lineStr)
		if err != nil {
			return nil, fmt.Errorf("invalid line number: %v", err)
		}
		bp.Line = line
	} else {
		// Event type breakpoint
		bp.Type = EventTypeBreakpoint
		bp.EventType = location
	}

	bm.breakpoints = append(bm.breakpoints, bp)
	return bp, nil
}

// GetBreakpoints returns all breakpoints
func (bm *BreakpointManager) GetBreakpoints() []*Breakpoint {
	return bm.breakpoints
}

// RemoveBreakpoint removes a breakpoint by ID
func (bm *BreakpointManager) RemoveBreakpoint(id int) error {
	for i, bp := range bm.breakpoints {
		if bp.ID == id {
			bm.breakpoints = append(bm.breakpoints[:i], bm.breakpoints[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("breakpoint %d not found", id)
}

// EnableBreakpoint enables a breakpoint by ID
func (bm *BreakpointManager) EnableBreakpoint(id int) error {
	for _, bp := range bm.breakpoints {
		if bp.ID == id {
			bp.Enabled = true
			return nil
		}
	}
	return fmt.Errorf("breakpoint %d not found", id)
}

// DisableBreakpoint disables a breakpoint by ID
func (bm *BreakpointManager) DisableBreakpoint(id int) error {
	for _, bp := range bm.breakpoints {
		if bp.ID == id {
			bp.Enabled = false
			return nil
		}
	}
	return fmt.Errorf("breakpoint %d not found", id)
}

// CheckBreakpoint checks if a breakpoint should be hit
func (bm *BreakpointManager) CheckBreakpoint(details string, eventType string) bool {
	for _, bp := range bm.breakpoints {
		if !bp.Enabled {
			continue
		}

		if bp.Type == EventTypeBreakpoint && bp.EventType == eventType {
			return true
		}

		if bp.Type == FunctionBreakpoint && strings.Contains(details, bp.Function) {
			return true
		}
	}
	return false
}

// AddWatchpoint adds a watchpoint for an expression
func (bm *BreakpointManager) AddWatchpoint(expression string, watchType BreakpointType) (*Breakpoint, error) {
	if watchType != WatchpointRead && watchType != WatchpointWrite && watchType != WatchpointReadWrite {
		return nil, fmt.Errorf("invalid watchpoint type")
	}

	bp := &Breakpoint{
		ID:         bm.nextID,
		Type:       watchType,
		Expression: expression,
		Enabled:    true,
	}
	bm.nextID++

	bm.breakpoints = append(bm.breakpoints, bp)
	return bp, nil
}

// GetWatchpoints returns all watchpoints
func (bm *BreakpointManager) GetWatchpoints() []*Breakpoint {
	watchpoints := make([]*Breakpoint, 0)
	for _, bp := range bm.breakpoints {
		if bp.Type == WatchpointRead || bp.Type == WatchpointWrite || bp.Type == WatchpointReadWrite {
			watchpoints = append(watchpoints, bp)
		}
	}
	return watchpoints
}
