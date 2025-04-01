package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/debugger"
	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// testFunction is the function we'll debug
func testFunction() int {
	_, file, line, _ := runtime.Caller(0)
	instrumentation.FuncEntry("testFunction", file, line)
	defer func() {
		_, file, line, _ := runtime.Caller(0)
		instrumentation.FuncExit("testFunction", file, line)
	}()

	x := 42
	_, file, line, _ = runtime.Caller(0)
	instrumentation.RecordStatement("testFunction", file, line, "x = 42")

	y := x * 2
	_, file, line, _ = runtime.Caller(0)
	instrumentation.RecordStatement("testFunction", file, line, "y = x * 2")

	return y
}

// The main function coordinates the debugger and replayer
func main() {
	// Initialize instrumentation for the main function
	_, file, line, _ := runtime.Caller(0)
	instrumentation.FuncEntry("main", file, line)
	defer func() {
		_, file, line, _ := runtime.Caller(0)
		instrumentation.FuncExit("main", file, line)
	}()

	if len(os.Args) < 2 {
		fmt.Println("Usage: chrono <program>")
		os.Exit(1)
	}

	targetPath := os.Args[1]
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		fmt.Printf("Failed to get absolute path: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("ChronoGo Time-Travel Debugger")
	fmt.Println("-----------------------------")

	// Initialize recorder with a clean instance
	rec := recorder.NewInMemoryRecorder()
	instrumentation.InitInstrumentation(rec)

	// Create a replayer
	replayer := replay.NewBasicReplayer()

	// Try to initialize Delve debugger
	delveDebugger, delveErr := debugger.NewDelveDebugger(absPath)

	// If we have a debugger, preemptively set a breakpoint at testFunction
	if delveErr == nil {
		fmt.Println("Delve debugger initialized. Setting breakpoint in testFunction...")

		// Set breakpoint at the x := 42 line in testFunction
		bp, err := delveDebugger.SetBreakpoint("cmd/chrono/main.go", 23)
		if err != nil {
			fmt.Printf("Warning: Failed to set breakpoint: %v\n", err)
		} else {
			fmt.Printf("Set breakpoint at %s:%d\n", bp.File, bp.Line)
		}
	}

	// Execute the function we'll debug
	fmt.Println("\nRunning testFunction()...")
	result := testFunction()
	fmt.Printf("Function result: %d\n", result)

	// Record key points in the main function
	_, file, line, _ = runtime.Caller(0)
	instrumentation.RecordStatement("main", file, line, "After testFunction call")

	// Get recorded events and load them into the replayer
	events := rec.GetEvents()
	fmt.Printf("\nRecorded %d events:\n", len(events))
	for i, e := range events {
		fmt.Printf("[%d] %s: %s\n", i,
			e.Timestamp.Format(time.RFC3339),
			e.Details)
	}
	fmt.Println() // Empty line for readability

	replayer.LoadEvents(events)

	// Start the appropriate CLI (with or without Delve)
	if delveErr != nil {
		fmt.Printf("Warning: Failed to initialize Delve debugger: %v\n", delveErr)
		fmt.Println("Running in replay-only mode (no live debugging)")
		cli := debugger.NewCLI(replayer)
		cli.Start()
	} else {
		fmt.Println("Delve debugger initialized successfully")
		cli := debugger.NewCLIWithDelve(replayer, delveDebugger)
		cli.Start()
	}
}
