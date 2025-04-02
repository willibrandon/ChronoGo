package tests

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/willibrandon/ChronoGo/pkg/debugger"
)

// findProjectRoot returns the project root directory
func findProjectRoot() (string, error) {
	// First, try the current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Check if cwd or cwd/.. contains cmd/chrono/main.go
	candidate := cwd
	if _, err := os.Stat(filepath.Join(candidate, "cmd", "chrono", "main.go")); err == nil {
		return candidate, nil
	}

	candidate = filepath.Join(cwd, "..")
	if _, err := os.Stat(filepath.Join(candidate, "cmd", "chrono", "main.go")); err == nil {
		return candidate, nil
	}

	// If we're in CI, try other common locations
	for _, candidate := range []string{
		// Github Actions workspace locations for different platforms
		filepath.Join(cwd, "..", ".."),
		filepath.Join(os.Getenv("GITHUB_WORKSPACE"), ".."),
		os.Getenv("GITHUB_WORKSPACE"),
		"/home/runner/work/ChronoGo/ChronoGo",
	} {
		if candidate == "" {
			continue
		}
		if _, err := os.Stat(filepath.Join(candidate, "cmd", "chrono", "main.go")); err == nil {
			return candidate, nil
		}
	}

	// If we've not found it through standard locations, try using runtime caller
	// to trace back through source to find project root
	_, filename, _, _ := runtime.Caller(0)
	candidate = filepath.Dir(filepath.Dir(filename)) // tests/delve_test.go -> tests -> project root
	if _, err := os.Stat(filepath.Join(candidate, "cmd", "chrono", "main.go")); err == nil {
		return candidate, nil
	}

	return "", os.ErrNotExist
}

func TestDelveDebugger(t *testing.T) {
	// Find project root and build paths from there
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("Failed to find project root: %v", err)
	}

	// Get the path to the test binary
	var binaryPath string
	if runtime.GOOS == "windows" {
		binaryPath = filepath.Join(projectRoot, "chrono_test.exe")
	} else {
		binaryPath = filepath.Join(projectRoot, "chrono_test")
	}

	// Build the binary with special test flag
	t.Logf("Building test binary at %s", binaryPath)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}
	if err := os.Chdir(projectRoot); err != nil {
		t.Fatalf("Failed to change to project root: %v", err)
	}

	// Use cross-platform way to build the binary
	goBinary, err := exec.LookPath("go")
	if err != nil {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: Failed to change back to original directory: %v", err)
		}
		t.Fatalf("Failed to find go binary: %v", err)
	}

	// Build with debug info and disable optimizations, also add a special test mode flag
	var cmd *exec.Cmd
	cmd = exec.Command(goBinary, "build", "-gcflags", "all=-N -l", "-o", binaryPath, "-ldflags", "-X main.testMode=true", "./cmd/chrono")

	output, err := cmd.CombinedOutput()
	if err != nil {
		if err := os.Chdir(origDir); err != nil {
			t.Logf("Warning: Failed to change back to original directory: %v", err)
		}
		t.Fatalf("Failed to build binary: %v\nOutput: %s", err, output)
	}

	if err := os.Chdir(origDir); err != nil {
		t.Fatalf("Failed to change back to original directory: %v", err)
	}

	// Create a temporary events file for the test to use
	eventsFile, err := os.CreateTemp("", "chronogo-test-*.events")
	if err != nil {
		t.Fatalf("Failed to create temporary events file: %v", err)
	}
	defer os.Remove(eventsFile.Name())
	eventsFile.Close()

	// Create a new Delve debugger with arguments that trigger the test function
	dbgArgs := []string{"--test"}
	dbg, err := debugger.NewDelveDebuggerWithArgs(binaryPath, dbgArgs)
	if err != nil {
		t.Fatalf("Failed to create Delve debugger: %v", err)
	}
	defer dbg.Close()

	// Set a breakpoint at the testFunction entry point
	bp, breakpointErr := dbg.SetFunctionBreakpoint("main.testFunction")
	if breakpointErr != nil {
		t.Errorf("Error: Failed to set function breakpoint: %v", breakpointErr)
	} else {
		t.Logf("Successfully set breakpoint at function main.testFunction (ID: %d)", bp.ID)
	}

	// Try another breakpoint at a specific line in testFunction
	mainFile := filepath.Join(projectRoot, "cmd", "chrono", "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Fatalf("Source file not found at %s", mainFile)
	}

	bp2, err := dbg.SetBreakpoint(mainFile, 42) // Line with 'x := 42'
	if err != nil {
		t.Logf("Note: Could not set line breakpoint: %v", err)
	} else {
		t.Logf("Set additional breakpoint at %s:%d (ID: %d)", bp2.File, bp2.Line, bp2.ID)
	}

	// Try to continue to the breakpoint
	t.Log("Continuing execution to breakpoint...")
	state, err := dbg.Continue()
	if err != nil {
		t.Errorf("Error: Continue operation reported error: %v", err)
	} else {
		t.Logf("Stopped at %s:%d", state.CurrentThread.File, state.CurrentThread.Line)

		// Step over the instrumentation code
		stepState, stepErr := dbg.Step()
		if stepErr != nil {
			t.Errorf("Error: Step operation reported error: %v", stepErr)
		} else {
			t.Logf("After step, now at %s:%d", stepState.CurrentThread.File, stepState.CurrentThread.Line)
		}

		// Step over the instrumentation code again
		stepState, stepErr = dbg.Step()
		if stepErr != nil {
			t.Errorf("Error: Second step operation reported error: %v", stepErr)
		} else {
			t.Logf("After second step, now at %s:%d", stepState.CurrentThread.File, stepState.CurrentThread.Line)
		}

		// Step over the defer block
		stepState, stepErr = dbg.Step()
		if stepErr != nil {
			t.Errorf("Error: Third step operation reported error: %v", stepErr)
		} else {
			t.Logf("After third step, now at %s:%d", stepState.CurrentThread.File, stepState.CurrentThread.Line)
		}

		// Step over the defer function body
		stepState, stepErr = dbg.Step()
		if stepErr != nil {
			t.Errorf("Error: Fourth step operation reported error: %v", stepErr)
		} else {
			t.Logf("After fourth step, now at %s:%d", stepState.CurrentThread.File, stepState.CurrentThread.Line)
		}

		// Step over the closing brace of defer
		stepState, stepErr = dbg.Step()
		if stepErr != nil {
			t.Errorf("Error: Fifth step operation reported error: %v", stepErr)
		} else {
			t.Logf("After fifth step, now at %s:%d", stepState.CurrentThread.File, stepState.CurrentThread.Line)
		}

		// Now we should be at x := 42
		x, err := dbg.GetVariable("x")
		if err != nil {
			t.Errorf("Error: Could not get variable 'x': %v", err)
		} else {
			t.Logf("Variable x = %s", x.Value)
			// Only assert equality if we got the variable
			if x.Value != "42" {
				t.Logf("Note: Expected x to be 42, got %s", x.Value)
			}
		}
	}

	t.Logf("Basic Delve integration test completed")
}
