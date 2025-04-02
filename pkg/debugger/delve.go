package debugger

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/go-delve/delve/service/rpc2"
)

// DelveDebugger wraps a Delve RPC client session, managing the underlying dlv process
type DelveDebugger struct {
	client    *rpc2.RPCClient
	target    string    // Target binary path
	dlvCmd    *exec.Cmd // The running 'dlv exec' command
	dlvListen string    // The address dlv is listening on (e.g., "localhost:12345")
}

// findFreePort finds an available TCP port on localhost
func findFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// NewDelveDebugger launches a Delve headless server for the target and connects via RPC
func NewDelveDebugger(targetPath string) (*DelveDebugger, error) {
	// Convert to absolute path
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for target %s: %v", targetPath, err)
	}

	// Find an available port for Delve to listen on
	port, err := findFreePort()
	if err != nil {
		return nil, fmt.Errorf("failed to find free port for delve: %v", err)
	}
	dlvListenAddr := "localhost:" + strconv.Itoa(port)

	// Construct the dlv exec command
	// Ensure dlv executable is in PATH or provide full path
	cmdArgs := []string{
		"exec", absPath,
		"--headless",
		"--listen=" + dlvListenAddr,
		"--api-version=2",
		"--accept-multiclient",
	}
	dlvCmd := exec.Command("dlv", cmdArgs...)

	// Prevent dlv from creating a new console window on Windows
	dlvCmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// Start the Delve headless server
	if err := dlvCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start delve process: %v", err)
	}
	fmt.Printf("Started Delve headless server for %s on %s (PID: %d)\n", absPath, dlvListenAddr, dlvCmd.Process.Pid)

	// Wait a moment for the server to initialize
	time.Sleep(500 * time.Millisecond) // Adjust timing if needed

	// Connect the RPC client
	client := rpc2.NewClient(dlvListenAddr)

	// Simple connection check
	if _, err := client.GetState(); err != nil {
		// If connection fails, attempt to kill the dlv process we started
		_ = dlvCmd.Process.Kill()
		_, _ = dlvCmd.Process.Wait() // Wait to clean up zombie process
		return nil, fmt.Errorf("failed to connect RPC client to delve server at %s: %v", dlvListenAddr, err)
	}

	fmt.Printf("Connected RPC client to Delve headless server at %s\n", dlvListenAddr)

	return &DelveDebugger{
		client:    client,
		target:    absPath,
		dlvCmd:    dlvCmd,
		dlvListen: dlvListenAddr,
	}, nil
}

// SetBreakpoint sets a breakpoint at the specified location using RPC
func (d *DelveDebugger) SetBreakpoint(file string, line int) (*api.Breakpoint, error) {
	// Normalize file path (for Windows compatibility)
	file = filepath.ToSlash(file)

	// Try to find the exact file and line
	bp := &api.Breakpoint{
		File: file,
		Line: line,
	}

	// First try setting the breakpoint directly
	createdBp, err := d.client.CreateBreakpoint(bp)
	if err == nil {
		return createdBp, nil
	}

	// If exact match failed, try finding closest line with a statement
	// This can help when debugging code with optimizations or go run issues
	if strings.Contains(err.Error(), "could not find statement") {
		// Try searching for the source file
		sources, locErr := d.client.ListSources(file)
		if locErr == nil && len(sources) > 0 {
			// Try looking for other nearby lines
			for offset := 1; offset <= 5; offset++ {
				// Try line + offset
				nearbyBp := &api.Breakpoint{
					File: file,
					Line: line + offset,
				}
				if nearByCreated, nearbyErr := d.client.CreateBreakpoint(nearbyBp); nearbyErr == nil {
					// Found a valid line nearby - remove it immediately since we're just testing
					_, _ = d.client.ClearBreakpoint(nearByCreated.ID)
					return nil, fmt.Errorf("%v\nTry line %d instead, which contains an executable statement",
						err, line+offset)
				}

				// Try line - offset if it's positive
				if line-offset > 0 {
					nearbyBp = &api.Breakpoint{
						File: file,
						Line: line - offset,
					}
					if nearByCreated, nearbyErr := d.client.CreateBreakpoint(nearbyBp); nearbyErr == nil {
						// Found a valid line nearby - remove it immediately since we're just testing
						_, _ = d.client.ClearBreakpoint(nearByCreated.ID)
						return nil, fmt.Errorf("%v\nTry line %d instead, which contains an executable statement",
							err, line-offset)
					}
				}
			}
		}
	}

	// If we couldn't find any alternatives, return the original error
	return nil, err
}

// ClearBreakpoint removes a breakpoint by its ID using RPC
func (d *DelveDebugger) ClearBreakpoint(id int) error {
	_, err := d.client.ClearBreakpoint(id)
	return err
}

// Continue resumes execution until the next breakpoint using RPC
func (d *DelveDebugger) Continue() (*api.DebuggerState, error) {
	stateChan := d.client.Continue()
	state := <-stateChan
	if state.Err != nil {
		return nil, state.Err
	}
	return state, nil
}

// Step executes a single instruction using RPC
func (d *DelveDebugger) Step() (*api.DebuggerState, error) {
	state, err := d.client.Next()
	if err != nil {
		return nil, fmt.Errorf("step command failed: %v", err)
	}
	if state.Err != nil {
		return nil, state.Err
	}
	return state, nil
}

// StepOut steps out of the current function using RPC
func (d *DelveDebugger) StepOut() (*api.DebuggerState, error) {
	state, err := d.client.StepOut()
	if err != nil {
		return nil, fmt.Errorf("step out command failed: %v", err)
	}
	if state.Err != nil {
		return nil, state.Err
	}
	return state, nil
}

// GetVariable retrieves the value of a variable using RPC
func (d *DelveDebugger) GetVariable(name string) (*api.Variable, error) {
	state, err := d.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}
	if state.CurrentThread == nil {
		return nil, fmt.Errorf("no current thread available")
	}

	// Create evaluation scope based on current thread
	scope := api.EvalScope{
		GoroutineID: state.CurrentThread.GoroutineID,
		Frame:       0,
	}

	// Standard load config
	cfg := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	}

	// Try multiple approaches to find the variable

	// 1. Direct evaluation
	v, err := d.client.EvalVariable(scope, name, cfg)
	if err == nil {
		return v, nil
	}

	// 2. Alternate syntax (.name)
	v, err = d.client.EvalVariable(scope, fmt.Sprintf(".%s", name), cfg)
	if err == nil {
		return v, nil
	}

	// 3. Try manually listing local variables
	locals, err := d.client.ListLocalVariables(scope, cfg)
	if err == nil {
		for _, local := range locals {
			if local.Name == name {
				return &api.Variable{
					Name:  local.Name,
					Value: local.Value,
					Type:  local.Type,
				}, nil
			}
		}
	}

	// 4. Check if it's a function argument
	args, err := d.client.ListFunctionArgs(scope, cfg)
	if err == nil {
		for _, arg := range args {
			if arg.Name == name {
				return &api.Variable{
					Name:  arg.Name,
					Value: arg.Value,
					Type:  arg.Type,
				}, nil
			}
		}
	}

	// If all attempts fail, provide a clearer message
	return nil, fmt.Errorf("failed to evaluate variable '%s': could not find symbol value for %s", name, name)
}

// ListGoroutines returns all active goroutines using RPC
func (d *DelveDebugger) ListGoroutines() ([]*api.Goroutine, error) {
	goroutines, _, err := d.client.ListGoroutines(0, 0)
	if err != nil {
		return nil, err
	}
	return goroutines, nil
}

// Close terminates the connection and the Delve process
func (d *DelveDebugger) Close() error {
	var closeErr error
	if d.client != nil {
		// Detach from the process (kill=false is often problematic here, let Kill handle it)
		// _, _ = d.client.Detach(false) // Try detaching gracefully first? Might hang.
		if err := d.client.Disconnect(false); err != nil { // Disconnect the RPC client
			fmt.Printf("Error disconnecting Delve client: %v\n", err)
			closeErr = fmt.Errorf("failed to disconnect delve client: %v", err)
		}
		d.client = nil
	}
	if d.dlvCmd != nil && d.dlvCmd.Process != nil {
		fmt.Printf("Attempting to terminate Delve process (PID: %d)...\n", d.dlvCmd.Process.Pid)
		if err := d.dlvCmd.Process.Kill(); err != nil {
			// If already exited, it's not an error we need to report upwards usually.
			if err.Error() != "os: process already finished" {
				fmt.Printf("Error killing delve process %d: %v\n", d.dlvCmd.Process.Pid, err)
				closeErr = fmt.Errorf("failed to kill delve process: %v", err)
			}
		}
		// Wait for the process to release resources
		_, waitErr := d.dlvCmd.Process.Wait()
		if waitErr != nil && waitErr.Error() != "os: process already finished" && !isWaitAlreadyExited(waitErr) {
			fmt.Printf("Error waiting for delve process %d: %v\n", d.dlvCmd.Process.Pid, waitErr)
			// Append wait error if kill error didn't happen or was minor
			if closeErr == nil {
				closeErr = fmt.Errorf("failed to wait for delve process: %v", waitErr)
			}
		}
		fmt.Printf("Delve process (PID: %d) terminated.\n", d.dlvCmd.Process.Pid)
		d.dlvCmd = nil
	}
	return closeErr
}

// Helper to check for specific Wait error on Windows
func isWaitAlreadyExited(err error) bool {
	if e, ok := err.(*exec.ExitError); ok {
		if status, ok := e.Sys().(syscall.WaitStatus); ok {
			// Check if the exit code indicates the process was already gone
			// This status code might vary or need refinement
			return status.ExitStatus() == -1 // Common for already exited process on Windows?
		}
	}
	return false
}

// SetWatchpoint sets a watchpoint on a variable or address
func (d *DelveDebugger) SetWatchpoint(expr string, readFlag, writeFlag bool) (*api.Breakpoint, error) {
	// Get current state to determine goroutine/frame context
	state, err := d.client.GetState()
	if err != nil {
		return nil, fmt.Errorf("failed to get state: %v", err)
	}

	// Create evaluation scope
	scope := api.EvalScope{
		GoroutineID: -1, // Use default goroutine
		Frame:       0,  // Use current frame
	}

	// If we have a current thread, use its goroutine
	if state.CurrentThread != nil && state.CurrentThread.GoroutineID > 0 {
		scope.GoroutineID = state.CurrentThread.GoroutineID
	}

	// Standard load config
	cfg := api.LoadConfig{
		FollowPointers:     true,
		MaxVariableRecurse: 1,
		MaxStringLen:       64,
		MaxArrayValues:     64,
		MaxStructFields:    -1,
	}

	// Try to evaluate the expression to get the address
	v, err := d.client.EvalVariable(scope, expr, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate expression '%s': %v", expr, err)
	}

	// Create a basic breakpoint at the variable's address
	bp := &api.Breakpoint{
		Addr: v.Addr,
	}

	// Set the breakpoint conditions based on read/write flags
	// This is a simplified approach - Delve's API might provide
	// more direct support for watchpoints depending on the version
	if readFlag && writeFlag {
		bp.Cond = fmt.Sprintf("(read || write) to %s", expr)
	} else if readFlag {
		bp.Cond = fmt.Sprintf("read of %s", expr)
	} else if writeFlag {
		bp.Cond = fmt.Sprintf("write to %s", expr)
	} else {
		return nil, fmt.Errorf("at least one of read or write flag must be set")
	}

	// Set the watchpoint using Delve API
	return d.client.CreateBreakpoint(bp)
}
