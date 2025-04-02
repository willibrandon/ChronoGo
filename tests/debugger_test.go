package tests

import (
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/debugger"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

func TestBreakpointManager(t *testing.T) {
	bm := debugger.NewBreakpointManager()

	// Test adding different types of breakpoints
	tests := []struct {
		name     string
		location string
		wantErr  bool
	}{
		{"Location breakpoint", "main.go:42", false},
		{"Function breakpoint", "func:testFunction", false},
		{"Event breakpoint", "FunctionEntry", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bp, err := bm.AddBreakpoint(tt.location)
			if (err != nil) != tt.wantErr {
				t.Errorf("AddBreakpoint() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if bp == nil {
				t.Error("AddBreakpoint() returned nil breakpoint")
				return
			}
			if !bp.Enabled {
				t.Error("New breakpoint should be enabled by default")
			}
		})
	}

	// Test breakpoint management
	bp, _ := bm.AddBreakpoint("test.go:10")
	bpID := bp.ID

	// Test disable/enable
	if err := bm.DisableBreakpoint(bpID); err != nil {
		t.Errorf("DisableBreakpoint() error = %v", err)
	}

	// Find the breakpoint by ID
	bps := bm.GetBreakpoints()
	var foundBP *debugger.Breakpoint
	for _, bp := range bps {
		if bp.ID == bpID {
			foundBP = bp
			break
		}
	}

	if foundBP == nil {
		t.Fatalf("Could not find breakpoint with ID %d", bpID)
	}

	if foundBP.Enabled {
		t.Error("Breakpoint should be disabled")
	}

	if err := bm.EnableBreakpoint(bpID); err != nil {
		t.Errorf("EnableBreakpoint() error = %v", err)
	}

	// Find the breakpoint again
	bps = bm.GetBreakpoints()
	foundBP = nil
	for _, bp := range bps {
		if bp.ID == bpID {
			foundBP = bp
			break
		}
	}

	if foundBP == nil {
		t.Fatalf("Could not find breakpoint with ID %d", bpID)
	}

	if !foundBP.Enabled {
		t.Error("Breakpoint should be enabled")
	}

	// Test removal
	if err := bm.RemoveBreakpoint(bpID); err != nil {
		t.Errorf("RemoveBreakpoint() error = %v", err)
	}
	bps = bm.GetBreakpoints()
	if len(bps) != 3 { // 3 from the first test cases
		t.Errorf("Expected 3 breakpoints after removal, got %d", len(bps))
	}
}

func TestBreakpointHitting(t *testing.T) {
	bm := debugger.NewBreakpointManager()

	// Add breakpoints
	_, err := bm.AddBreakpoint("func:testFunction")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	_, err = bm.AddBreakpoint("FunctionEntry")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	// Create some test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering main",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(100 * time.Millisecond),
			Type:      recorder.FuncEntry,
			Details:   "Entering testFunction",
		},
	}

	// Check if breakpoints are hit for each event
	for _, event := range events {
		if !bm.CheckBreakpoint(event.Details, event.Type.String()) {
			t.Errorf("Should hit breakpoint for event: %s - %s", event.Type, event.Details)
		}
	}
}

func TestCLICreation(t *testing.T) {
	// Create a replayer with some test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering main",
		},
	}

	replayer := replay.NewBasicReplayer()
	err := replayer.LoadEvents(events)
	if err != nil {
		t.Errorf("Failed to load events: %v", err)
	}

	// Create CLI
	cli := debugger.NewCLI(replayer)
	if cli == nil {
		t.Error("NewCLI() returned nil")
	}
}
