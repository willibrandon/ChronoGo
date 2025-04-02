package debugger

import (
	"testing"
)

func TestNewBreakpointManager(t *testing.T) {
	bm := NewBreakpointManager()
	if bm == nil {
		t.Fatal("NewBreakpointManager returned nil")
	}

	if bm.nextID != 1 {
		t.Errorf("Expected nextID to be 1, got %d", bm.nextID)
	}

	if len(bm.breakpoints) != 0 {
		t.Errorf("Expected 0 breakpoints, got %d", len(bm.breakpoints))
	}
}

func TestAddBreakpoint(t *testing.T) {
	bm := NewBreakpointManager()

	testCases := []struct {
		name     string
		location string
		wantType BreakpointType
		wantErr  bool
	}{
		{
			name:     "Location breakpoint",
			location: "test.go:42",
			wantType: LocationBreakpoint,
			wantErr:  false,
		},
		{
			name:     "Function breakpoint",
			location: "func:main",
			wantType: FunctionBreakpoint,
			wantErr:  false,
		},
		{
			name:     "Event type breakpoint",
			location: "FunctionEntry",
			wantType: EventTypeBreakpoint,
			wantErr:  false,
		},
		{
			name:     "Windows path location",
			location: "C:/path/to/file.go:42",
			wantType: LocationBreakpoint,
			wantErr:  false,
		},
		{
			name:     "Invalid location",
			location: "invalid",
			wantType: EventTypeBreakpoint, // Defaults to this
			wantErr:  false,
		},
	}

	for i, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bp, err := bm.AddBreakpoint(tc.location)

			if tc.wantErr && err == nil {
				t.Errorf("Expected error, got nil")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if bp == nil {
				t.Fatal("AddBreakpoint returned nil breakpoint")
			}

			if bp.Type != tc.wantType {
				t.Errorf("Expected type %v, got %v", tc.wantType, bp.Type)
			}

			if bp.ID != i+1 {
				t.Errorf("Expected ID %d, got %d", i+1, bp.ID)
			}

			if !bp.Enabled {
				t.Error("Breakpoint should be enabled by default")
			}
		})
	}

	// Check that all breakpoints are stored
	if len(bm.breakpoints) != len(testCases) {
		t.Errorf("Expected %d breakpoints, got %d", len(testCases), len(bm.breakpoints))
	}
}

func TestGetBreakpoints(t *testing.T) {
	bm := NewBreakpointManager()

	// Add a few breakpoints
	_, err := bm.AddBreakpoint("test.go:10")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	_, err = bm.AddBreakpoint("func:main")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	_, err = bm.AddBreakpoint("VarAssignment")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	breakpoints := bm.GetBreakpoints()

	if len(breakpoints) != 3 {
		t.Errorf("Expected 3 breakpoints, got %d", len(breakpoints))
	}
}

func TestRemoveBreakpoint(t *testing.T) {
	bm := NewBreakpointManager()

	// Add breakpoints
	bp1, _ := bm.AddBreakpoint("test.go:10")
	bp2, _ := bm.AddBreakpoint("func:main")

	// Remove the first breakpoint
	err := bm.RemoveBreakpoint(bp1.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check remaining breakpoints
	breakpoints := bm.GetBreakpoints()
	if len(breakpoints) != 1 {
		t.Errorf("Expected 1 breakpoint, got %d", len(breakpoints))
	}

	if breakpoints[0].ID != bp2.ID {
		t.Errorf("Expected remaining breakpoint ID %d, got %d", bp2.ID, breakpoints[0].ID)
	}

	// Try to remove a non-existent breakpoint
	err = bm.RemoveBreakpoint(999)
	if err == nil {
		t.Error("Expected error when removing non-existent breakpoint, got nil")
	}
}

func TestEnableDisableBreakpoint(t *testing.T) {
	bm := NewBreakpointManager()

	// Add a breakpoint
	bp, _ := bm.AddBreakpoint("test.go:10")

	// Disable it
	err := bm.DisableBreakpoint(bp.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check it's disabled
	breakpoints := bm.GetBreakpoints()
	if breakpoints[0].Enabled {
		t.Error("Breakpoint should be disabled")
	}

	// Enable it
	err = bm.EnableBreakpoint(bp.ID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Check it's enabled
	breakpoints = bm.GetBreakpoints()
	if !breakpoints[0].Enabled {
		t.Error("Breakpoint should be enabled")
	}

	// Try to enable/disable a non-existent breakpoint
	err = bm.EnableBreakpoint(999)
	if err == nil {
		t.Error("Expected error when enabling non-existent breakpoint, got nil")
	}

	err = bm.DisableBreakpoint(999)
	if err == nil {
		t.Error("Expected error when disabling non-existent breakpoint, got nil")
	}
}

func TestCheckBreakpoint(t *testing.T) {
	bm := NewBreakpointManager()

	// Add different types of breakpoints
	_, err := bm.AddBreakpoint("func:testFunc")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	_, err = bm.AddBreakpoint("FuncEntry")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	// Disable the first breakpoint
	err = bm.DisableBreakpoint(1)
	if err != nil {
		t.Fatalf("Failed to disable breakpoint: %v", err)
	}

	testCases := []struct {
		details   string
		eventType string
		want      bool
	}{
		{
			details:   "Entering testFunc",
			eventType: "FuncEntry",
			want:      true, // Should match the second breakpoint (FuncEntry)
		},
		{
			details:   "Entering testFunc",
			eventType: "FuncExit",
			want:      false, // Different event type
		},
		{
			details:   "Entering otherFunc",
			eventType: "FuncEntry",
			want:      true, // Still matches FuncEntry even though function differs
		},
	}

	for _, tc := range testCases {
		hit := bm.CheckBreakpoint(tc.details, tc.eventType)
		if hit != tc.want {
			t.Errorf("CheckBreakpoint(%q, %q) = %v, want %v", tc.details, tc.eventType, hit, tc.want)
		}
	}
}

func TestAddWatchpoint(t *testing.T) {
	bm := NewBreakpointManager()

	// Test adding read watchpoint
	wp, err := bm.AddWatchpoint("x", WatchpointRead)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if wp.Type != WatchpointRead {
		t.Errorf("Expected type %v, got %v", WatchpointRead, wp.Type)
	}

	if wp.Expression != "x" {
		t.Errorf("Expected expression %q, got %q", "x", wp.Expression)
	}

	// Test adding write watchpoint
	wp2, err := bm.AddWatchpoint("y", WatchpointWrite)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if wp2.Type != WatchpointWrite {
		t.Errorf("Expected type %v, got %v", WatchpointWrite, wp2.Type)
	}

	// Test adding read/write watchpoint
	wp3, err := bm.AddWatchpoint("z", WatchpointReadWrite)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if wp3.Type != WatchpointReadWrite {
		t.Errorf("Expected type %v, got %v", WatchpointReadWrite, wp3.Type)
	}

	// Test invalid watchpoint type
	_, err = bm.AddWatchpoint("a", LocationBreakpoint)
	if err == nil {
		t.Error("Expected error for invalid watchpoint type, got nil")
	}
}

func TestGetWatchpoints(t *testing.T) {
	bm := NewBreakpointManager()

	// Add various breakpoint types
	_, err := bm.AddBreakpoint("test.go:10")
	if err != nil {
		t.Fatalf("Failed to add breakpoint: %v", err)
	}

	_, err = bm.AddWatchpoint("x", WatchpointRead)
	if err != nil {
		t.Fatalf("Failed to add watchpoint: %v", err)
	}

	_, err = bm.AddWatchpoint("y", WatchpointWrite)
	if err != nil {
		t.Fatalf("Failed to add watchpoint: %v", err)
	}

	watchpoints := bm.GetWatchpoints()

	if len(watchpoints) != 2 {
		t.Errorf("Expected 2 watchpoints, got %d", len(watchpoints))
	}
}
