package main

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/instrumentation"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
)

// Event generator for testing
func generateEvents(count int, r recorder.Recorder) {
	for i := 0; i < count; i++ {
		event := recorder.Event{
			ID:        int64(i),
			Timestamp: recorder.CurrentTime(),
			Type:      recorder.StatementExecution,
			Details:   fmt.Sprintf("Test event with some data %d", i),
			File:      "demo.go",
			Line:      i % 100,
			FuncName:  "generateEvents",
		}
		r.RecordEvent(event)
	}
}

// Helper function that may or may not be instrumented
func doSomething() {
	// This would normally contain code that gets instrumented
	time.Sleep(10 * time.Millisecond)
}

func demoCompression() {
	// Create temp directory for demo files
	tempDir, err := os.MkdirTemp("", "chronogo-compression-demo")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Create compressed file recorder
	compressedFile := filepath.Join(tempDir, "events_compressed.chrono")
	compressedOptions := recorder.FileRecorderOptions{
		CompressionType: recorder.ZstdCompression,
	}
	compressedRecorder, err := recorder.NewFileRecorderWithOptions(compressedFile, compressedOptions)
	if err != nil {
		fmt.Printf("Error creating compressed recorder: %v\n", err)
		return
	}

	// Create uncompressed file recorder
	uncompressedFile := filepath.Join(tempDir, "events_uncompressed.chrono")
	uncompressedOptions := recorder.FileRecorderOptions{
		CompressionType: recorder.NoCompression,
	}
	uncompressedRecorder, err := recorder.NewFileRecorderWithOptions(uncompressedFile, uncompressedOptions)
	if err != nil {
		fmt.Printf("Error creating uncompressed recorder: %v\n", err)
		return
	}

	fmt.Println("Generating events...")
	const eventCount = 10000

	// Generate events for both recorders
	generateEvents(eventCount, compressedRecorder)
	generateEvents(eventCount, uncompressedRecorder)

	// Close recorders to ensure data is flushed
	compressedRecorder.Close()
	uncompressedRecorder.Close()

	// Get file info
	compressedInfo, err := os.Stat(compressedFile)
	if err != nil {
		fmt.Printf("Error getting compressed file info: %v\n", err)
		return
	}

	uncompressedInfo, err := os.Stat(uncompressedFile)
	if err != nil {
		fmt.Printf("Error getting uncompressed file info: %v\n", err)
		return
	}

	// Display results
	compressedSize := compressedInfo.Size()
	uncompressedSize := uncompressedInfo.Size()
	savingsPercent := (1.0 - float64(compressedSize)/float64(uncompressedSize)) * 100

	fmt.Printf("\nCompression Results:\n")
	fmt.Printf("Uncompressed file size: %d bytes\n", uncompressedSize)
	fmt.Printf("Compressed file size:   %d bytes\n", compressedSize)
	fmt.Printf("Space savings:          %.2f%%\n", savingsPercent)

	// Verify we can read back the events
	fmt.Println("\nVerifying compressed data integrity...")
	compressedRecorderRead, err := recorder.NewFileRecorderWithOptions(compressedFile, compressedOptions)
	if err != nil {
		fmt.Printf("Error opening compressed file for reading: %v\n", err)
		return
	}
	defer compressedRecorderRead.Close()

	events := compressedRecorderRead.GetEvents()
	fmt.Printf("Successfully read back %d events from compressed file\n", len(events))
}

func demoSnapshots() {
	// Create temp directory for demo files
	tempDir, err := os.MkdirTemp("", "chronogo-snapshots-demo")
	if err != nil {
		fmt.Printf("Error creating temp directory: %v\n", err)
		return
	}
	defer os.RemoveAll(tempDir)

	// Save the original snapshot interval
	originalInterval := recorder.SnapshotInterval

	// Set a custom interval for the demo
	recorder.SnapshotInterval = 1000

	// Create file recorder with custom snapshot intervals
	snapshotFile := filepath.Join(tempDir, "events_with_snapshots.chrono")
	snapshotRecorder, err := recorder.NewFileRecorder(snapshotFile)
	if err != nil {
		fmt.Printf("Error creating recorder: %v\n", err)
		return
	}

	fmt.Println("Generating events with snapshots...")
	const eventCount = 10000
	generateEvents(eventCount, snapshotRecorder)
	snapshotRecorder.Close()

	// Open the file for reading to check snapshots
	readRecorder, err := recorder.NewFileRecorder(snapshotFile)
	if err != nil {
		fmt.Printf("Error opening file for reading: %v\n", err)
		return
	}
	defer readRecorder.Close()

	// Get all events
	events := readRecorder.GetEvents()

	// Count snapshots
	snapshots := 0
	for _, event := range events {
		if event.Type == recorder.SnapshotEvent {
			snapshots++
		}
	}

	fmt.Printf("\nSnapshot Results:\n")
	fmt.Printf("Total events generated: %d\n", eventCount)
	fmt.Printf("Snapshot interval:      %d events\n", recorder.SnapshotInterval)
	fmt.Printf("Number of snapshots:    %d\n", snapshots)

	if snapshots > 0 {
		fmt.Println("\nFirst few snapshot positions:")
		count := 0
		for i, event := range events {
			if event.Type == recorder.SnapshotEvent && count < 5 {
				fmt.Printf("  Snapshot %d: at event index %d\n", count, i)
				count++
			}
		}
	}

	fmt.Println("\nWith snapshots, time-travel debugging is more efficient because")
	fmt.Println("the replayer can jump directly to the nearest snapshot rather than")
	fmt.Println("replaying from the beginning every time.")

	// Restore the original interval
	recorder.SnapshotInterval = originalInterval
}

func demoSelectiveInstrumentation() {
	fmt.Println("Selective Instrumentation Demo")
	fmt.Println("==============================")

	// Show current instrumentation settings
	opts := instrumentation.CurrentOptions
	fmt.Println("\nDefault settings:")
	fmt.Printf("  Instrumentation enabled: %t\n", opts.Enabled)
	fmt.Printf("  Standard library instrumented: %t\n", opts.InstrumentStdlib)
	fmt.Printf("  Included packages: %v\n", opts.IncludePackages)
	fmt.Printf("  Excluded packages: %v\n", opts.ExcludePackages)

	// Run a function that would normally be instrumented
	fmt.Println("\nRunning a function with default instrumentation...")
	doSomething()

	// Change options to exclude the current package
	currentPkg := getCurrentPackage()
	fmt.Printf("\nExcluding current package from instrumentation: %s\n", currentPkg)

	// Save the original options for restoration later
	originalOptions := opts

	newOpts := instrumentation.InstrumentationOptions{
		Enabled:          true,
		InstrumentStdlib: false,
		IncludePackages:  []string{},
		ExcludePackages:  []string{currentPkg},
	}
	instrumentation.SetInstrumentationOptions(newOpts)

	fmt.Println("Running the same function with the current package excluded...")
	doSomething() // Should not be instrumented

	// Restore default options
	instrumentation.SetInstrumentationOptions(originalOptions)

	fmt.Println("\nHow to set options via environment variables:")
	fmt.Println("  export CHRONOGO_ENABLED=true")
	fmt.Println("  export CHRONOGO_INSTRUMENT_STDLIB=false")
	fmt.Println("  export CHRONOGO_INSTRUMENT=github.com/yourusername/myapp,github.com/yourusername/utils")
	fmt.Println("  export CHRONOGO_EXCLUDE=github.com/yourusername/slow_package")
}

// Helper function to get the current package name
func getCurrentPackage() string {
	pc, _, _, ok := runtime.Caller(1)
	if !ok {
		return "unknown"
	}

	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "unknown"
	}

	name := fn.Name()
	lastSlash := strings.LastIndexByte(name, '/')
	if lastSlash < 0 {
		lastSlash = 0
	}

	if period := strings.IndexByte(name[lastSlash:], '.'); period >= 0 {
		name = name[:lastSlash+period]
	}

	return name
}

// Helper function since Go 1.20 doesn't have built-in min for ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func main() {
	// Set random seed for Go 1.20 since Go 1.20 still requires explicit seeding
	rand.Seed(time.Now().UnixNano())

	// Check if we have an argument
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run demo.go [compression|snapshots|selective]")
		os.Exit(1)
	}

	// Run the requested demo
	switch os.Args[1] {
	case "compression":
		demoCompression()
	case "snapshots":
		demoSnapshots()
	case "selective":
		demoSelectiveInstrumentation()
	default:
		fmt.Printf("Unknown demo: %s\n", os.Args[1])
		fmt.Println("Available demos: compression, snapshots, selective")
	}
}
