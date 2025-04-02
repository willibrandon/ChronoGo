package debugger

import (
	"fmt"
	"net"
	"os/exec"
	"path/filepath"
	"reflect"
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

// NewDelveDebuggerWithArgs launches a Delve headless server for the target with the given command line arguments and connects via RPC
func NewDelveDebuggerWithArgs(targetPath string, args []string) (*DelveDebugger, error) {
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

	// Construct the dlv exec command with args
	// Ensure dlv executable is in PATH or provide full path
	cmdArgs := []string{
		"exec", absPath,
		"--headless",
		"--listen=" + dlvListenAddr,
		"--api-version=2",
		"--accept-multiclient",
	}

	// Only add the '--' separator if we have args to pass
	if len(args) > 0 {
		cmdArgs = append(cmdArgs, "--")
		cmdArgs = append(cmdArgs, args...)
	}

	dlvCmd := exec.Command("dlv", cmdArgs...)

	// Platform-specific process attributes are set in setupProcAttr function
	setupProcAttr(dlvCmd)

	// Start the Delve headless server
	if err := dlvCmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start delve process: %v", err)
	}
	fmt.Printf("Started Delve headless server for %s on %s (PID: %d) with args: %v\n",
		absPath, dlvListenAddr, dlvCmd.Process.Pid, args)

	// Wait a moment for the server to initialize - longer time for testing
	time.Sleep(1000 * time.Millisecond)

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

// NewDelveDebugger launches a Delve headless server for the target and connects via RPC
func NewDelveDebugger(targetPath string) (*DelveDebugger, error) {
	return NewDelveDebuggerWithArgs(targetPath, nil)
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

	// If exact match failed, try more strategies

	// 1. Try finding closest line with a statement
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
					fmt.Printf("Successfully set breakpoint at alternative line %d instead of %d\n",
						line+offset, line)
					return nearByCreated, nil
				}

				// Try line - offset if it's positive
				if line-offset > 0 {
					nearbyBp = &api.Breakpoint{
						File: file,
						Line: line - offset,
					}
					if nearByCreated, nearbyErr := d.client.CreateBreakpoint(nearbyBp); nearbyErr == nil {
						fmt.Printf("Successfully set breakpoint at alternative line %d instead of %d\n",
							line-offset, line)
						return nearByCreated, nil
					}
				}
			}

			// If we still haven't found a suitable line, provide more detailed suggestions
			return nil, fmt.Errorf("%v\nTry one of these lines instead, which might contain executable statements: %d, %d, %d",
				err, line+1, line+2, line-1)
		}
	}

	// 2. Check for file path discrepancies (common in Go module builds)
	if strings.Contains(err.Error(), "no file") || strings.Contains(err.Error(), "does not exist") {
		// Get list of all sources
		sources, listErr := d.client.ListSources("")
		if listErr != nil {
			return nil, fmt.Errorf("failed to list sources: %v (original error: %v)", listErr, err)
		}

		// Check for basename matches
		baseName := filepath.Base(file)
		var matchingFiles []string
		for _, src := range sources {
			if filepath.Base(src) == baseName {
				matchingFiles = append(matchingFiles, src)
			}
		}

		// Try setting breakpoint with matching file paths
		for _, matchFile := range matchingFiles {
			alternativeBp := &api.Breakpoint{
				File: matchFile,
				Line: line,
			}
			if altCreated, altErr := d.client.CreateBreakpoint(alternativeBp); altErr == nil {
				fmt.Printf("Successfully set breakpoint using alternative path %s\n", matchFile)
				return altCreated, nil
			}
		}

		if len(matchingFiles) > 0 {
			return nil, fmt.Errorf("%v\nTried alternative files: %v, but couldn't set breakpoint",
				err, matchingFiles)
		}
	}

	// 3. Try setting a function breakpoint if we can determine the function name
	// This would require parsing the source file or using other methods to determine
	// which function contains the given line number

	// If all strategies failed, return the original error
	return nil, fmt.Errorf("could not set breakpoint at %s:%d: %v", file, line, err)
}

// SetFunctionBreakpoint sets a breakpoint at a function
func (d *DelveDebugger) SetFunctionBreakpoint(funcName string) (*api.Breakpoint, error) {
	// Create a breakpoint specification targeting the function
	bp := &api.Breakpoint{
		FunctionName: funcName,
	}

	// Try to create the breakpoint
	createdBp, err := d.client.CreateBreakpoint(bp)
	if err == nil {
		return createdBp, nil
	}

	// If the exact function name didn't work, try with package prefix variations
	if strings.Contains(err.Error(), "could not find function") {
		// Try with common package prefix variations
		prefixes := []string{
			"main.", // Most common
			"runtime.",
			"github.com/",
		}

		for _, prefix := range prefixes {
			if !strings.HasPrefix(funcName, prefix) {
				altFuncName := prefix + funcName
				altBp := &api.Breakpoint{
					FunctionName: altFuncName,
				}

				if altCreated, altErr := d.client.CreateBreakpoint(altBp); altErr == nil {
					fmt.Printf("Successfully set breakpoint at function %s\n", altFuncName)
					return altCreated, nil
				}
			}
		}

		// If we still haven't found the function, try to list available functions
		funcs, _ := d.client.ListFunctions(funcName, 10) // Limit to 10 matching functions
		if len(funcs) > 0 {
			suggestions := funcs
			if len(suggestions) > 5 {
				suggestions = suggestions[:5] // Limit to 5 suggestions
			}

			return nil, fmt.Errorf("%v\nDid you mean one of these functions?\n%s",
				err, strings.Join(suggestions, "\n"))
		}
	}

	return nil, fmt.Errorf("could not set breakpoint at function %s: %v", funcName, err)
}

// SetConditionalBreakpoint sets a breakpoint with a condition
func (d *DelveDebugger) SetConditionalBreakpoint(file string, line int, condition string) (*api.Breakpoint, error) {
	// Normalize file path
	file = filepath.ToSlash(file)

	// Create the breakpoint with condition
	bp := &api.Breakpoint{
		File: file,
		Line: line,
		Cond: condition,
	}

	// Try to create the breakpoint
	createdBp, err := d.client.CreateBreakpoint(bp)
	if err != nil {
		// If the breakpoint location is valid but the condition is invalid
		if strings.Contains(err.Error(), "condition") {
			return nil, fmt.Errorf("invalid condition '%s': %v", condition, err)
		}

		// Otherwise, try the regular breakpoint setting logic
		regularBp, regularErr := d.SetBreakpoint(file, line)
		if regularErr != nil {
			return nil, fmt.Errorf("could not set conditional breakpoint: %v", regularErr)
		}

		// If we could set a regular breakpoint, try to amend it with the condition
		regularBp.Cond = condition
		if amendErr := d.client.AmendBreakpoint(regularBp); amendErr != nil {
			// Clean up the regular breakpoint if we can't make it conditional
			_, _ = d.client.ClearBreakpoint(regularBp.ID)
			return nil, fmt.Errorf("could set breakpoint but failed to add condition: %v", amendErr)
		}

		return regularBp, nil
	}

	return createdBp, nil
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

	// Customize load config based on variable type detection
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
		// Detect if this is a complex type and customize loading
		return d.loadComplexVariable(v, scope)
	}

	// 2. Alternate syntax (.name)
	v, err = d.client.EvalVariable(scope, fmt.Sprintf(".%s", name), cfg)
	if err == nil {
		return d.loadComplexVariable(v, scope)
	}

	// 3. Try manually listing local variables
	locals, err := d.client.ListLocalVariables(scope, cfg)
	if err == nil {
		for _, local := range locals {
			if local.Name == name {
				v := &api.Variable{
					Name:  local.Name,
					Value: local.Value,
					Type:  local.Type,
				}
				return d.loadComplexVariable(v, scope)
			}
		}
	}

	// 4. Check if it's a function argument
	args, err := d.client.ListFunctionArgs(scope, cfg)
	if err == nil {
		for _, arg := range args {
			if arg.Name == name {
				v := &api.Variable{
					Name:  arg.Name,
					Value: arg.Value,
					Type:  arg.Type,
				}
				return d.loadComplexVariable(v, scope)
			}
		}
	}

	// If all attempts fail, provide a clearer message
	return nil, fmt.Errorf("failed to evaluate variable '%s': could not find symbol value for %s", name, name)
}

// loadComplexVariable provides enhanced loading for complex variable types
func (d *DelveDebugger) loadComplexVariable(v *api.Variable, scope api.EvalScope) (*api.Variable, error) {
	// Already loaded simple types can be returned as-is
	if v.Kind == reflect.Bool || v.Kind == reflect.Int || v.Kind == reflect.Float64 ||
		v.Kind == reflect.String || v.Kind == reflect.Float32 || v.Kind == reflect.Int8 ||
		v.Kind == reflect.Int16 || v.Kind == reflect.Int32 || v.Kind == reflect.Int64 ||
		v.Kind == reflect.Uint || v.Kind == reflect.Uint8 || v.Kind == reflect.Uint16 ||
		v.Kind == reflect.Uint32 || v.Kind == reflect.Uint64 || v.Kind == reflect.Uintptr {
		return v, nil
	}

	// Create type-specific loading configurations
	var cfg api.LoadConfig

	switch v.Kind {
	case reflect.Struct:
		// For structs, load all fields
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     0,  // Don't load arrays within structs by default
			MaxStructFields:    -1, // Load all struct fields
		}
	case reflect.Slice, reflect.Array:
		// For slices/arrays, load more elements but limit recursion
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 0, // Don't follow pointers in array elements by default
			MaxStringLen:       32,
			MaxArrayValues:     100, // Show more array elements
			MaxStructFields:    -1,
		}
	case reflect.Map:
		// For maps, load more key/values
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       32,
			MaxArrayValues:     100, // More map entries
			MaxStructFields:    -1,
		}
	case reflect.Ptr, reflect.Interface:
		// For pointers, increase recursion
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 2, // Follow deeper
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		}
	case reflect.Chan:
		// For channels, try to examine buffer and structure
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     32, // Show buffered channel elements
			MaxStructFields:    -1,
		}
	default:
		// Default config for other types
		cfg = api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		}
	}

	// Re-evaluate with the type-specific config
	return d.client.EvalVariable(scope, v.Name, cfg)
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
