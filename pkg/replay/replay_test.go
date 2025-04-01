package replay

import (
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

func TestBasicReplayerLoading(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Create some test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.FuncEntry,
			Details:   "Entering function1",
			File:      "test.go",
			Line:      10,
			FuncName:  "function1",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(time.Millisecond * 10),
			Type:      recorder.FuncExit,
			Details:   "Exiting function1",
			File:      "test.go",
			Line:      20,
			FuncName:  "function1",
		},
	}

	// Load events
	err := replayer.LoadEvents(events)
	if err != nil {
		t.Fatalf("Failed to load events: %v", err)
	}

	// Check current index
	if replayer.CurrentIndex() != -1 {
		t.Errorf("Expected current index to be -1, got %d", replayer.CurrentIndex())
	}

	// Check events
	loadedEvents := replayer.Events()
	if len(loadedEvents) != len(events) {
		t.Errorf("Expected %d events, got %d", len(events), len(loadedEvents))
	}
}

func TestReplayToEventIndex(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Create some test events
	events := []recorder.Event{
		{ID: 1, Type: recorder.FuncEntry, Details: "Event 1"},
		{ID: 2, Type: recorder.VarAssignment, Details: "Event 2"},
		{ID: 3, Type: recorder.FuncExit, Details: "Event 3"},
	}

	// Load events
	replayer.LoadEvents(events)

	// Replay to index 1
	err := replayer.ReplayToEventIndex(1)
	if err != nil {
		t.Fatalf("Failed to replay to event index: %v", err)
	}

	// Check current index
	if replayer.CurrentIndex() != 1 {
		t.Errorf("Expected current index to be 1, got %d", replayer.CurrentIndex())
	}

	// Test invalid index (negative)
	err = replayer.ReplayToEventIndex(-5)
	if err != nil {
		t.Errorf("ReplayToEventIndex with negative index should not return error")
	}

	// Test invalid index (beyond array)
	err = replayer.ReplayToEventIndex(100)
	if err != nil {
		t.Errorf("ReplayToEventIndex with out-of-bounds index should not return error")
	}
}

func TestStepBackward(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Create some test events
	events := []recorder.Event{
		{ID: 1, Type: recorder.FuncEntry, Details: "Event 1"},
		{ID: 2, Type: recorder.VarAssignment, Details: "Event 2"},
		{ID: 3, Type: recorder.FuncExit, Details: "Event 3"},
	}

	// Load events
	replayer.LoadEvents(events)

	// Set to index 2
	replayer.ReplayToEventIndex(2)

	// Step back
	newIdx, err := replayer.StepBackward(replayer.CurrentIndex())
	if err != nil {
		t.Fatalf("Failed to step backward: %v", err)
	}

	// Check new index
	if newIdx != 1 {
		t.Errorf("Expected new index to be 1, got %d", newIdx)
	}

	// Step back again
	newIdx, err = replayer.StepBackward(replayer.CurrentIndex())
	if err != nil {
		t.Fatalf("Failed to step backward: %v", err)
	}

	// Check new index
	if newIdx != 0 {
		t.Errorf("Expected new index to be 0, got %d", newIdx)
	}

	// Step back at beginning should fail
	_, err = replayer.StepBackward(replayer.CurrentIndex())
	if err == nil {
		t.Errorf("Expected error when stepping back at beginning, got nil")
	}
}

func TestReplayUntilBreakpoint(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Create some test events
	events := []recorder.Event{
		{ID: 1, Type: recorder.FuncEntry, Details: "Entering function1", FuncName: "function1"},
		{ID: 2, Type: recorder.VarAssignment, Details: "var x = 5", FuncName: "function1"},
		{ID: 3, Type: recorder.FuncExit, Details: "Exiting function1", FuncName: "function1"},
		{ID: 4, Type: recorder.FuncEntry, Details: "Entering function2", FuncName: "function2"},
		{ID: 5, Type: recorder.FuncExit, Details: "Exiting function2", FuncName: "function2"},
	}

	// Load events
	replayer.LoadEvents(events)

	// Create a breakpoint check for function2 entry
	breakpointCheck := func(event recorder.Event) bool {
		return event.Type == recorder.FuncEntry && event.FuncName == "function2"
	}

	// Replay until breakpoint
	err := replayer.ReplayUntilBreakpoint(breakpointCheck)
	if err != nil {
		t.Fatalf("Failed to replay until breakpoint: %v", err)
	}

	// Check current index (should be 3, which is event ID 4)
	if replayer.CurrentIndex() != 3 {
		t.Errorf("Expected current index to be 3, got %d", replayer.CurrentIndex())
	}
}

func TestConcurrencyEvents(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Create some concurrency test events
	events := []recorder.Event{
		{
			ID:        1,
			Timestamp: time.Now(),
			Type:      recorder.GoroutineSwitch,
			Details:   "Goroutine 2 created",
		},
		{
			ID:        2,
			Timestamp: time.Now().Add(time.Millisecond * 10),
			Type:      recorder.GoroutineSwitch,
			Details:   "Goroutine switch from 1 to 2",
		},
		{
			ID:        3,
			Timestamp: time.Now().Add(time.Millisecond * 20),
			Type:      recorder.ChannelOperation,
			Details:   "Channel 1: send by goroutine 2",
		},
		{
			ID:        4,
			Timestamp: time.Now().Add(time.Millisecond * 30),
			Type:      recorder.GoroutineSwitch,
			Details:   "Goroutine switch from 2 to 1",
		},
		{
			ID:        5,
			Timestamp: time.Now().Add(time.Millisecond * 40),
			Type:      recorder.ChannelOperation,
			Details:   "Channel 1: receive by goroutine 1",
		},
		{
			ID:        6,
			Timestamp: time.Now().Add(time.Millisecond * 50),
			Type:      recorder.SyncOperation,
			Details:   "Mutex 1: locked by goroutine 1",
		},
		{
			ID:        7,
			Timestamp: time.Now().Add(time.Millisecond * 60),
			Type:      recorder.SyncOperation,
			Details:   "Mutex 1: unlocked by goroutine 1",
		},
	}

	// Load events
	replayer.LoadEvents(events)

	// Replay all events to test concurrency handling
	err := replayer.ReplayForward()
	if err != nil {
		t.Fatalf("Failed to replay events: %v", err)
	}

	// Check current index
	if replayer.CurrentIndex() != len(events)-1 {
		t.Errorf("Expected current index to be %d, got %d", len(events)-1, replayer.CurrentIndex())
	}
}

func TestReplayerWithNoEvents(t *testing.T) {
	// Create a replayer
	replayer := NewBasicReplayer()

	// Test replay with no events
	err := replayer.ReplayForward()
	if err != nil {
		t.Errorf("ReplayForward with no events should not return error, got: %v", err)
	}

	// Test replay until breakpoint with no events
	err = replayer.ReplayUntilBreakpoint(func(event recorder.Event) bool { return true })
	if err != nil {
		t.Errorf("ReplayUntilBreakpoint with no events should not return error, got: %v", err)
	}
}
