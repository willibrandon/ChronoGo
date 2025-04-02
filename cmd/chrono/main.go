package main

import (
	"bufio"
	"encoding/json"
	"flag"
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

// Define a custom usage function to show detailed help
func printUsage() {
	fmt.Println("ChronoGo Time-Travel Debugger")
	fmt.Println("-----------------------------")
	fmt.Println("Usage: chrono [options] <program>")
	fmt.Println("\nOptions:")
	fmt.Println("  -events <file>    Specify events file path (default: chronogo.events)")
	fmt.Println("  -replay           Run in replay mode only (no execution)")
	fmt.Println("  -help             Show this help message")
	fmt.Println("\nExamples:")
	fmt.Println("  chrono myapp                        # Debug myapp with default settings")
	fmt.Println("  chrono -events custom.log myapp     # Debug with custom events file")
	fmt.Println("  chrono -replay -events saved.log    # Replay events from saved.log")
	fmt.Println("\nReplay Mode Commands:")
	fmt.Println("  c, continue       Continue execution until the next breakpoint")
	fmt.Println("  s, step           Step forward one event")
	fmt.Println("  b, backstep       Step backward one event")
	fmt.Println("  q, quit           Exit the debugger")
	fmt.Println("  show              Show the current execution state")
	fmt.Println("  help              Display available commands")
}

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

func loadEventsFromFile(filePath string) ([]recorder.Event, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening events file: %v", err)
	}
	defer file.Close()

	var events []recorder.Event
	scanner := bufio.NewScanner(file)

	// Increase scanner buffer size for larger JSON lines
	const maxCapacity = 512 * 1024 // 512KB
	buf := make([]byte, maxCapacity)
	scanner.Buffer(buf, maxCapacity)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if len(line) == 0 {
			continue // Skip empty lines
		}

		var event recorder.Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			fmt.Printf("Warning: Could not parse event on line %d: %v\n", lineNum, err)
			continue
		}
		events = append(events, event)
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading events file: %v", err)
	}

	fmt.Printf("Successfully parsed %d events from file\n", len(events))
	return events, nil
}

// The main function coordinates the debugger and replayer
func main() {
	// Set custom usage function for better help
	flag.Usage = printUsage

	// Parse command line flags
	eventsFileFlag := flag.String("events", "chronogo.events", "Path to the events file")
	replayModeFlag := flag.Bool("replay", false, "Run in replay mode only (no execution)")
	helpFlag := flag.Bool("help", false, "Show help message")
	flag.Parse()

	// If help flag is explicitly set, show usage and exit
	if *helpFlag {
		printUsage()
		return
	}

	fmt.Println("ChronoGo Time-Travel Debugger")
	fmt.Println("-----------------------------")

	// Check if replay mode was explicitly requested
	if *replayModeFlag {
		if _, err := os.Stat(*eventsFileFlag); err != nil {
			fmt.Printf("Error: Cannot find events file '%s' for replay\n", *eventsFileFlag)
			os.Exit(1)
		}

		fmt.Printf("Loading events from: %s\n", *eventsFileFlag)
		events, err := loadEventsFromFile(*eventsFileFlag)
		if err != nil {
			fmt.Printf("Error loading events: %v\n", err)
			os.Exit(1)
		}

		if len(events) == 0 {
			fmt.Println("Error: No events found in the specified file")
			os.Exit(1)
		}

		fmt.Printf("Loaded %d events. Entering replay mode...\n", len(events))
		replayer := replay.NewBasicReplayer()
		if err := replayer.LoadEvents(events); err != nil {
			fmt.Printf("Error loading events: %v\n", err)
		}
		cli := debugger.NewCLI(replayer)
		cli.Start()
		return
	}

	// Check for the default events file first (what test.go writes to)
	defaultEventsFile := "chronogo.events"
	customEventsFile := *eventsFileFlag

	// If the user specified a custom events file and it differs from the default
	if defaultEventsFile != customEventsFile {
		// Check if the default file exists (which would have been written by test.go)
		if _, err := os.Stat(defaultEventsFile); err == nil {
			// If we found the default file but we want a custom name, copy it
			fmt.Printf("Found default events file: %s\n", defaultEventsFile)
			fmt.Printf("Copying to requested events file: %s\n", customEventsFile)

			// Read the default events file
			data, err := os.ReadFile(defaultEventsFile)
			if err != nil {
				fmt.Printf("Error reading default events file: %v\n", err)
			} else {
				// Write to the custom file
				err = os.WriteFile(customEventsFile, data, 0644)
				if err != nil {
					fmt.Printf("Error writing to custom events file: %v\n", err)
				}
			}
		}
	}

	// Check if the events file exists (either the default or custom one)
	if _, err := os.Stat(customEventsFile); err == nil {
		fmt.Printf("Found events file: %s\n", customEventsFile)
		events, err := loadEventsFromFile(customEventsFile)
		if err != nil {
			fmt.Printf("Error loading events: %v\n", err)
		} else if len(events) > 0 {
			fmt.Printf("Loaded %d events. Entering replay mode...\n", len(events))

			// Initialize replayer with loaded events
			replayer := replay.NewBasicReplayer()
			if err := replayer.LoadEvents(events); err != nil {
				fmt.Printf("Error loading events: %v\n", err)
			}

			// Start CLI in replay mode
			cli := debugger.NewCLI(replayer)
			cli.Start()
			return
		} else {
			fmt.Println("Events file exists but contains no valid events.")
		}
	}

	// If we reach here, either no events file or it's empty
	// Check if we have a program to debug
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Usage: chrono [options] <program>")
		fmt.Println("\nOptions:")
		fmt.Println("  -events <file>    Specify events file path (default: chronogo.events)")
		fmt.Println("  -replay           Run in replay mode only (no execution)")
		fmt.Println("\nExamples:")
		fmt.Println("  chrono myapp               # Debug myapp with default settings")
		fmt.Println("  chrono -events custom.log myapp  # Debug with custom events file")
		fmt.Println("  chrono -replay -events saved.log # Replay events from saved.log")
		return
	}

	targetPath := args[0]
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		fmt.Printf("Failed to get absolute path: %v\n", err)
		os.Exit(1)
	}

	// Initialize instrumentation for the main function
	_, file, line, _ := runtime.Caller(0)
	instrumentation.FuncEntry("main", file, line)
	defer func() {
		_, file, line, _ := runtime.Caller(0)
		instrumentation.FuncExit("main", file, line)
	}()

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

	// Optionally save events to the specified file
	if len(events) > 0 {
		fileRec, err := recorder.NewFileRecorder(customEventsFile)
		if err == nil {
			for _, e := range events {
				if err := fileRec.RecordEvent(e); err != nil {
					fmt.Printf("Warning: Failed to record event: %v\n", err)
				}
			}
			fileRec.Close()
			fmt.Printf("Saved %d events to %s\n", len(events), customEventsFile)
		} else {
			fmt.Printf("Warning: Failed to save events to %s: %v\n", customEventsFile, err)
		}
	}

	if err := replayer.LoadEvents(events); err != nil {
		fmt.Printf("Error loading events: %v\n", err)
	}

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
