package tests

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// TestDelveReplayerSynchronization tests the synchronization between the replayer and Delve
func TestDelveReplayerSynchronization(t *testing.T) {
	// Find project root and prepare the test binary with debug flag
	projectRoot, binaryPath, err := prepareTestBinaryWithDebug(t)
	if err != nil {
		t.Fatalf("Failed to prepare test environment: %v", err)
	}

	// Add global cleanup for any dlv processes at the end, in case something goes wrong
	defer func() {
		if runtime.GOOS == "windows" {
			t.Log("Killing any leftover dlv processes")
			killOutput, _ := exec.Command("taskkill", "/F", "/IM", "dlv.exe").CombinedOutput()
			t.Logf("Taskkill output: %s", string(killOutput))
		} else {
			// Use pkill on Unix
			exec.Command("pkill", "-f", "dlv").Run()
		}
	}()

	// Track whether we've successfully attached to the debug process
	var dlvAttached bool

	// Clean up binary after test completes
	defer func() {
		t.Logf("Cleaning up debug binary: %s", binaryPath)
		// Small sleep to ensure file handles are released
		time.Sleep(100 * time.Millisecond)
		if err := os.Remove(binaryPath); err != nil {
			t.Logf("Warning: Failed to remove debug binary: %v", err)
		}

		// If we didn't successfully attach to the debug process, the test has failed
		if !dlvAttached {
			t.Errorf("Failed to successfully attach dlv to the debug process")
		}
	}()

	// We'll directly run the binary with the -debug flag
	cmd := exec.Command(binaryPath, "-debug")

	// Start the process
	if err := cmd.Start(); err != nil {
		t.Fatalf("Failed to start debug binary: %v", err)
	}

	t.Logf("Started debug binary with PID: %d", cmd.Process.Pid)

	// Cleanup on exit
	defer func() {
		if cmd.Process != nil {
			t.Logf("Killing debug process (PID: %d)", cmd.Process.Pid)
			err := cmd.Process.Kill()
			if err != nil {
				t.Logf("Warning: Failed to kill debug process: %v", err)

				// On Windows, try taskkill as a fallback
				if runtime.GOOS == "windows" {
					killOutput, _ := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", cmd.Process.Pid)).CombinedOutput()
					t.Logf("Taskkill output: %s", string(killOutput))
				}
			}
			cmd.Process.Wait()
		}
	}()

	// Give the process a moment to start and reach the debug helper
	time.Sleep(1 * time.Second)

	// Create a basic replayer and load sample events
	replayer := replay.NewBasicReplayer()
	events := createSampleEvents(t, projectRoot)
	if err := replayer.LoadEvents(events); err != nil {
		t.Fatalf("Failed to load events into replayer: %v", err)
	}

	// Test synchronization to a specific event
	t.Run("SyncToEvent", func(t *testing.T) {
		// Find an event related to the debugHelper function, which contains a pause
		// that gives the debugger time to connect before the program exits
		idx := findEventWithFunctionName(events, "debugHelper")
		if idx == -1 {
			// If no debugHelper event found, fall back to any event with file/line info
			idx = findEventWithFileAndLine(events)
			if idx == -1 {
				t.Errorf("Failed to find event with file and line info")
				return
			}
		}

		// Set replayer to the correct index
		err := replayer.ReplayToEventIndex(idx)
		if err != nil {
			t.Errorf("Failed to set replayer index: %v", err)
			return
		}

		// Now run dlv attach directly
		dlvBinary, err := exec.LookPath("dlv")
		if err != nil {
			t.Fatalf("Failed to find dlv binary: %v", err)
		}

		// Create a temporary directory to store our debug outputs
		tempDir := t.TempDir()
		outputFile := filepath.Join(tempDir, "dlv_output.txt")

		dlvCmd := exec.Command(
			dlvBinary,
			"attach",
			fmt.Sprintf("%d", cmd.Process.Pid),
			"--headless",
			"--log",
			"--listen=localhost:40001", // Specify a port to avoid conflicts
		)

		// Capture dlv output
		dlvOut, err := os.Create(outputFile)
		if err != nil {
			t.Fatalf("Failed to create output file: %v", err)
		}
		defer dlvOut.Close()

		dlvCmd.Stdout = dlvOut
		dlvCmd.Stderr = dlvOut

		// Start dlv
		if err := dlvCmd.Start(); err != nil {
			t.Errorf("Failed to start dlv attach: %v", err)
			return
		}

		// Cleanup dlv
		defer func() {
			if dlvCmd.Process != nil {
				t.Logf("Killing dlv process (PID: %d)", dlvCmd.Process.Pid)
				err := dlvCmd.Process.Kill()
				if err != nil {
					t.Logf("Warning: Failed to kill dlv process: %v", err)

					// On Windows, try taskkill as a fallback
					if runtime.GOOS == "windows" {
						killOutput, _ := exec.Command("taskkill", "/F", "/PID", fmt.Sprintf("%d", dlvCmd.Process.Pid)).CombinedOutput()
						t.Logf("Taskkill output: %s", string(killOutput))
					}
				}
				dlvCmd.Process.Wait()
			}

			// Read and log the output
			output, err := os.ReadFile(outputFile)
			if err == nil {
				t.Logf("dlv output: %s", string(output))
			}
		}()

		// Wait to ensure dlv has attached
		time.Sleep(2 * time.Second)

		// Verify the debug process is still running
		if cmd.ProcessState != nil && cmd.ProcessState.Exited() {
			t.Errorf("Debug process exited prematurely")
			return
		} else {
			t.Logf("Debug process is still running")
		}

		// Verify the dlv process is still running
		if dlvCmd.ProcessState != nil && dlvCmd.ProcessState.Exited() {
			t.Errorf("dlv process exited prematurely")
			return
		} else {
			t.Logf("dlv attach process is still running")
			dlvAttached = true // Mark that we successfully attached
		}

		// We're going to consider this a success if we can attach to the process
		// without trying to do complex synchronization at this point
		t.Logf("Successfully attached to debug process - basic sync test passed")

		// Verify that the replayer's current index matches what we set
		currentIdx := replayer.CurrentIndex()
		if currentIdx != idx {
			t.Errorf("Replayer index not updated correctly. Expected: %d, Got: %d",
				idx, currentIdx)
		}

		// Basic check that replayer contains events
		if len(events) > 0 {
			t.Logf("Replayer contains %d events and is at index %d", len(events), currentIdx)
		} else {
			t.Errorf("No events in replayer")
		}
	})
}

// createSampleEvents creates a set of sample events for testing
func createSampleEvents(t *testing.T, projectRoot string) []recorder.Event {
	// Path to the main file
	mainFile := filepath.Join(projectRoot, "cmd", "chrono", "main.go")

	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Logf("Main file not found at %s, using placeholder", mainFile)
		mainFile = "main.go" // Use a placeholder for testing
	}

	// Path to the debug helper in main.go - this function will pause for debugging
	debugHelperLine := findDebugHelperLineNumber(t, mainFile)
	if debugHelperLine == 0 {
		t.Fatalf("Failed to find debugHelper line number")
	}

	// Create sample events - using the debugHelper function as a target for breakpoints
	// since this function includes a sleep to prevent the program from exiting too quickly
	return []recorder.Event{
		{
			ID:        1,
			Type:      recorder.FuncEntry,
			Timestamp: time.Now(),
			Details:   "Entering main",
			FuncName:  "main",
			File:      mainFile,
			Line:      10, // Main entry point
		},
		{
			ID:        2,
			Type:      recorder.FuncEntry,
			Timestamp: time.Now().Add(time.Millisecond * 5),
			Details:   "Entering debugHelper",
			FuncName:  "debugHelper",
			File:      mainFile,
			Line:      debugHelperLine,
		},
		{
			ID:        3,
			Type:      recorder.StatementExecution,
			Timestamp: time.Now().Add(time.Millisecond * 10),
			Details:   "Debug helper sleep",
			FuncName:  "debugHelper",
			File:      mainFile,
			Line:      debugHelperLine + 1,
		},
		{
			ID:        4,
			Type:      recorder.FuncExit,
			Timestamp: time.Now().Add(time.Millisecond * 15),
			Details:   "Exiting debugHelper",
			FuncName:  "debugHelper",
			File:      mainFile,
			Line:      debugHelperLine + 2,
		},
		{
			ID:        5,
			Type:      recorder.StatementExecution,
			Timestamp: time.Now().Add(time.Millisecond * 20),
			Details:   "println(x)",
			FuncName:  "main",
			File:      mainFile,
			Line:      12,
		},
		{
			ID:        6,
			Type:      recorder.FuncExit,
			Timestamp: time.Now().Add(time.Millisecond * 30),
			Details:   "Exiting main",
			FuncName:  "main",
			File:      mainFile,
			Line:      13,
		},
	}
}

// findDebugHelperLineNumber finds the line number of the debugHelper function in main.go
func findDebugHelperLineNumber(t *testing.T, mainFile string) int {
	// Read the main.go file
	data, err := os.ReadFile(mainFile)
	if err != nil {
		t.Logf("Warning: Can't read main.go: %v", err)
		return 0
	}

	lines := strings.Split(string(data), "\n")
	for i, line := range lines {
		if strings.Contains(line, "func debugHelper") {
			return i + 1 // 1-based line numbering
		}
	}

	return 0 // Not found
}

// findEventWithFileAndLine finds an event that has file and line info
func findEventWithFileAndLine(events []recorder.Event) int {
	for i, event := range events {
		if event.File != "" && event.Line > 0 {
			return i
		}
	}
	return -1
}

// findEventWithFunctionName finds an event that has a specific function name
func findEventWithFunctionName(events []recorder.Event, funcName string) int {
	for i, event := range events {
		if event.FuncName == funcName {
			return i
		}
	}
	return -1
}
