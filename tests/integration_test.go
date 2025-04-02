package tests

import (
	"os"
	"strings"
	"testing"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// TestIntegrationWorkflow tests the entire workflow from instrumentation to recording to replay
func TestIntegrationWorkflow(t *testing.T) {
	// Step 1: Create a temporary file for event recording
	tempFile, err := os.CreateTemp("", "integration_test")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tempFile.Close()
	tempFilePath := tempFile.Name()
	defer os.Remove(tempFilePath)

	// Step 2: Create a recorder
	fileRecorder, err := recorder.NewFileRecorder(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to create file recorder: %v", err)
	}

	// Step 3: Initialize instrumentation
	instrumentation.InitInstrumentation(fileRecorder)

	// Step 4: Simulate instrumented function calls
	simulateInstrumentedCode(t, fileRecorder)

	// Step 5: Close the recorder
	err = fileRecorder.Close()
	if err != nil {
		t.Errorf("Failed to close recorder: %v", err)
	}

	// Step 6: Create a replayer to read back events
	replayer := replay.NewBasicReplayer()

	// Step 7: Open the file recorder for reading
	readRecorder, err := recorder.NewFileRecorder(tempFilePath)
	if err != nil {
		t.Fatalf("Failed to create read recorder: %v", err)
	}
	defer readRecorder.Close()

	// Step 8: Get the events
	events := readRecorder.GetEvents()
	if len(events) == 0 {
		t.Fatalf("No events recorded")
	}

	// Log the events for debugging
	t.Log("Starting replay:")
	for i, event := range events {
		t.Logf("Event %d: Type=%v FuncName=%s Details=%s",
			i+1,
			event.Type,
			event.FuncName,
			event.Details)
	}

	// Step 9: Load events into replayer
	err = replayer.LoadEvents(events)
	if err != nil {
		t.Errorf("Failed to load events into replayer: %v", err)
	}

	// Step 10: Use the replayer to detect breakpoints
	breakpointHit := false

	// Create a breakpoint checker function
	breakpointChecker := func(event recorder.Event) bool {
		t.Logf("Checking event: Type=%v FuncName=%s Details=%s",
			event.Type,
			event.FuncName,
			event.Details)

		// Check for function entry of testFunction, with more flexible matching
		if event.Type == recorder.FuncEntry {
			if event.FuncName == "testFunction" {
				t.Log("Breakpoint hit by function name!")
				breakpointHit = true
				return true
			}
			if event.Details != "" && strings.Contains(event.Details, "testFunction") {
				t.Log("Breakpoint hit by details containing function name!")
				breakpointHit = true
				return true
			}
		}
		return false
	}

	// Test the breakpoint using the replayer
	err = replayer.ReplayUntilBreakpoint(breakpointChecker)
	if err != nil {
		t.Errorf("Error during replay: %v", err)
	}

	if !breakpointHit {
		t.Error("Breakpoint was not hit during replay")
	}
}

// Helper function to simulate instrumented code execution
func simulateInstrumentedCode(t *testing.T, r recorder.Recorder) {
	// Simulate main function entry
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.FuncEntry,
		FuncName: "main",
		Details:  "Entering main function",
	}); err != nil {
		t.Errorf("Failed to record main entry event: %v", err)
	}

	// Simulate a function call with entry event
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.FuncEntry,
		FuncName: "testFunction",
		Details:  "Entering testFunction",
	}); err != nil {
		t.Errorf("Failed to record testFunction entry event: %v", err)
	}

	// Simulate variable assignment in the function
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.VarAssignment,
		FuncName: "testFunction",
		Details:  "y = 100",
	}); err != nil {
		t.Errorf("Failed to record variable assignment event: %v", err)
	}

	// Simulate function exit
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.FuncExit,
		FuncName: "testFunction",
		Details:  "Exiting testFunction",
	}); err != nil {
		t.Errorf("Failed to record function exit event: %v", err)
	}

	// Simulate goroutine creation
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.GoroutineSwitch,
		FuncName: "main",
		Details:  "Goroutine 2 created",
	}); err != nil {
		t.Errorf("Failed to record goroutine creation event: %v", err)
	}

	// Simulate variable assignment
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.VarAssignment,
		FuncName: "main",
		Details:  "x = 42",
	}); err != nil {
		t.Errorf("Failed to record variable assignment event: %v", err)
	}

	// Simulate channel operation
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.ChannelOperation,
		FuncName: "main",
		Details:  "Channel 1: send by goroutine 1, value: 42",
	}); err != nil {
		t.Errorf("Failed to record channel operation event: %v", err)
	}

	// Simulate main function exit
	if err := r.RecordEvent(recorder.Event{
		Type:     recorder.FuncExit,
		FuncName: "main",
		Details:  "Exiting main function",
	}); err != nil {
		t.Errorf("Failed to record main exit event: %v", err)
	}
}
