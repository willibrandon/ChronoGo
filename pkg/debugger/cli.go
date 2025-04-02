package debugger

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-delve/delve/service/api"
	"github.com/willibrandon/ChronoGo/pkg/recorder"
	"github.com/willibrandon/ChronoGo/pkg/replay"
)

// CLI represents the command-line interface for the debugger
type CLI struct {
	replayer  replay.Replayer
	debugger  *DelveDebugger
	running   bool
	bpManager *BreakpointManager
}

// NewCLI creates a new CLI instance
func NewCLI(replayer replay.Replayer) *CLI {
	return &CLI{
		replayer:  replayer,
		running:   false,
		bpManager: NewBreakpointManager(),
	}
}

// NewCLIWithDelve creates a new CLI instance with Delve integration
func NewCLIWithDelve(replayer replay.Replayer, dbg *DelveDebugger) *CLI {
	return &CLI{
		replayer:  replayer,
		debugger:  dbg,
		running:   false,
		bpManager: NewBreakpointManager(),
	}
}

// Start begins the command loop
func (c *CLI) Start() {
	c.running = true
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("ChronoGo Debugger CLI")
	if c.debugger != nil {
		fmt.Println("Delve integration enabled")
	}
	c.printHelp()

	for c.running {
		fmt.Print("(chrono) ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		c.handleCommand(input)
	}
}

// printHelp displays available commands
func (c *CLI) printHelp() {
	fmt.Println("\nAvailable commands:")
	fmt.Println("  continue (c)      - Continue execution")
	fmt.Println("  step (s)          - Step forward one event")
	fmt.Println("  backstep (b)      - Step backward one event")
	fmt.Println("  info (i)          - Show current execution state")

	if c.debugger != nil {
		fmt.Println("\nDelve debugging commands:")
		fmt.Println("  breakpoint (bp) <file:line> - Set a breakpoint")
		fmt.Println("  bp func:<funcname>  - Set a function breakpoint")
		fmt.Println("  bp <file:line> -c <cond> - Set a conditional breakpoint")
		fmt.Println("  list (l)        - List all breakpoints")
		fmt.Println("  print (p) <var> - Print value of a variable")
		fmt.Println("  goroutines (gr) - List all goroutines")
		fmt.Println("  watch (w) [-r|-w|-rw] <expr> - Set a watchpoint")
		fmt.Println("  bp remove <id>  - Remove a breakpoint")
		fmt.Println("  bp enable <id>  - Enable a breakpoint")
		fmt.Println("  bp disable <id> - Disable a breakpoint")
	}

	fmt.Println("\nGeneral commands:")
	fmt.Println("  help (h)          - Show this help message")
	fmt.Println("  quit (q)          - Exit the debugger")
}

// handleCommand processes user input
func (c *CLI) handleCommand(input string) {
	parts := strings.Fields(input)
	if len(parts) == 0 {
		return
	}

	cmd := parts[0]
	args := parts[1:]

	switch cmd {
	case "h", "help":
		c.printHelp()
	case "c", "continue":
		c.handleContinue()
	case "s", "step":
		c.handleStep()
	case "b", "backstep":
		c.handleBackstep()
	case "i", "info":
		c.handleInfo()
	case "q", "quit", "exit":
		c.running = false
		// Close delve if available
		if c.debugger != nil {
			c.debugger.Close()
		}
	// Delve-specific commands
	case "bp", "breakpoint":
		c.handleBreakpointCommand(args)
	case "l", "list":
		c.handleListBreakpoints()
	case "p", "print":
		c.handlePrintVariable(args)
	case "gr", "goroutines":
		c.handleListGoroutines()
	case "w", "watch":
		c.handleWatch(args)
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		c.printHelp()
	}
}

// handleBreakpointCommand handles all breakpoint-related commands
func (c *CLI) handleBreakpointCommand(args []string) {
	if c.debugger == nil {
		fmt.Println("Delve integration not enabled")
		return
	}

	if len(args) == 0 {
		// No args - show usage
		fmt.Println("Usage: breakpoint <file:line> or <command> [args]")
		fmt.Println("Commands: list, remove, enable, disable")
		fmt.Println("Function breakpoint: breakpoint func:<function_name>")
		fmt.Println("Conditional breakpoint: breakpoint <file:line> -c <condition>")
		return
	}

	command := args[0]

	switch command {
	case "list":
		c.handleListBreakpoints()
	case "remove":
		if len(args) < 2 {
			fmt.Println("Usage: bp remove <id>")
			return
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Invalid breakpoint ID: %v\n", err)
			return
		}

		// Remove from our manager
		err = c.bpManager.RemoveBreakpoint(id)
		if err != nil {
			fmt.Printf("Error removing breakpoint from manager: %v\n", err)
		}

		// Remove from Delve if available
		if c.debugger != nil {
			err = c.debugger.ClearBreakpoint(id)
			if err != nil {
				fmt.Printf("Error removing breakpoint from Delve: %v\n", err)
				return
			}
		}

		fmt.Printf("Removed breakpoint %d\n", id)
	case "enable":
		if len(args) < 2 {
			fmt.Println("Usage: bp enable <id>")
			return
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Invalid breakpoint ID: %v\n", err)
			return
		}

		// Enable in our manager
		err = c.bpManager.EnableBreakpoint(id)
		if err != nil {
			fmt.Printf("Error enabling breakpoint in manager: %v\n", err)
		}

		// Enable in Delve if available
		if c.debugger != nil {
			// Get current breakpoint
			bp, err := c.debugger.client.GetBreakpoint(id)
			if err != nil {
				fmt.Printf("Error getting breakpoint %d from Delve: %v\n", id, err)
				return
			}

			// Enable the breakpoint using Delve API
			bp.Disabled = false // Note: Delve uses Disabled rather than Enabled
			err = c.debugger.client.AmendBreakpoint(bp)
			if err != nil {
				fmt.Printf("Error enabling breakpoint %d in Delve: %v\n", id, err)
				return
			}
		}

		fmt.Printf("Enabled breakpoint %d\n", id)
	case "disable":
		if len(args) < 2 {
			fmt.Println("Usage: bp disable <id>")
			return
		}
		id, err := strconv.Atoi(args[1])
		if err != nil {
			fmt.Printf("Invalid breakpoint ID: %v\n", err)
			return
		}

		// Disable in our manager
		err = c.bpManager.DisableBreakpoint(id)
		if err != nil {
			fmt.Printf("Error disabling breakpoint in manager: %v\n", err)
		}

		// Disable in Delve if available
		if c.debugger != nil {
			// Get current breakpoint
			bp, err := c.debugger.client.GetBreakpoint(id)
			if err != nil {
				fmt.Printf("Error getting breakpoint %d from Delve: %v\n", id, err)
				return
			}

			// Disable the breakpoint using Delve API
			bp.Disabled = true // Note: Delve uses Disabled rather than Enabled
			err = c.debugger.client.AmendBreakpoint(bp)
			if err != nil {
				fmt.Printf("Error disabling breakpoint %d in Delve: %v\n", id, err)
				return
			}
		}

		fmt.Printf("Disabled breakpoint %d\n", id)
	default:
		// If not a command, treat as location
		c.handleBreakpoint(args)
	}
}

// formatEvent returns a string representation of an event
func (c *CLI) formatEvent(event recorder.Event) string {
	return fmt.Sprintf("[%s] Event %d: %s - %s",
		event.Timestamp.Format(time.RFC3339),
		event.ID,
		event.Type,
		event.Details)
}

// handleContinue resumes execution
func (c *CLI) handleContinue() {
	fmt.Println("Continuing execution...")

	// Create a breakpoint checker function
	breakpointChecker := func(event recorder.Event) bool {
		// Check if we have any breakpoints in the breakpoint manager
		for _, bp := range c.GetBreakpoints() {
			if !bp.Enabled {
				continue // Skip disabled breakpoints
			}

			// For file:line breakpoints, check if event's file and line match
			if bp.Type == LocationBreakpoint && event.File != "" && event.Line > 0 {
				// Normalize paths for comparison (convert backslashes to forward slashes)
				bpFile := strings.ReplaceAll(bp.File, "\\", "/")
				eventFile := strings.ReplaceAll(event.File, "\\", "/")

				// Normalize case for case-insensitive file systems (e.g., Windows)
				bpFile = strings.ToLower(bpFile)
				eventFile = strings.ToLower(eventFile)

				// Debug output for breakpoint comparison
				fmt.Printf("DEBUG: Checking breakpoint %s:%d against event at %s:%d\n",
					bpFile, bp.Line, eventFile, event.Line)

				if bpFile == eventFile && bp.Line == event.Line {
					fmt.Printf("HIT: Breakpoint at %s:%d\n", bp.File, bp.Line)
					return true
				}
			}

			// For function breakpoints, check event details
			if bp.Type == FunctionBreakpoint && event.Type == recorder.FuncEntry {
				if strings.Contains(event.Details, bp.Function) ||
					(event.FuncName != "" && strings.Contains(event.FuncName, bp.Function)) {
					return true
				}
			}

			// For event type breakpoints
			if bp.Type == EventTypeBreakpoint && event.Type.String() == bp.EventType {
				return true
			}
		}

		return false
	}

	// Continue in the replayer until breakpoint
	if err := c.replayer.ReplayUntilBreakpoint(breakpointChecker); err != nil {
		fmt.Printf("Error continuing execution: %v\n", err)
		return
	}

	// If Delve is available, also continue in the debugger
	if c.debugger != nil {
		state, err := c.debugger.Continue()
		if err != nil {
			fmt.Printf("Delve debugger error: %v\n", err)
		} else if state != nil {
			fmt.Printf("Debugger stopped at: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
		}
	}

	// Show current event
	events := c.replayer.Events()
	idx := c.replayer.CurrentIndex()
	if idx >= 0 && idx < len(events) {
		fmt.Printf("Current event: %s\n", c.formatEvent(events[idx]))
	}
}

// showCurrentVariables displays variables at the current execution point
func (c *CLI) showCurrentVariables() {
	if c.debugger == nil {
		return
	}

	// Try to get local variables
	state, err := c.debugger.client.GetState()
	if err != nil {
		fmt.Printf("Error getting state: %v\n", err)
		return
	}

	if state.CurrentThread == nil || state.CurrentThread.Function == nil {
		return
	}

	fmt.Printf("Current function: %s\n", state.CurrentThread.Function.Name())

	// Show x and y if we're in testFunction
	if strings.Contains(state.CurrentThread.Function.Name(), "testFunction") {
		// Try to get variable values
		vars, err := c.debugger.client.ListLocalVariables(api.EvalScope{
			GoroutineID: state.CurrentThread.GoroutineID,
			Frame:       0,
		}, api.LoadConfig{
			FollowPointers:     true,
			MaxVariableRecurse: 1,
			MaxStringLen:       64,
			MaxArrayValues:     64,
			MaxStructFields:    -1,
		})

		if err != nil {
			fmt.Printf("Error getting variables: %v\n", err)
			return
		}

		if len(vars) == 0 {
			fmt.Println("No local variables found")
			return
		}

		// Print variable info
		for _, v := range vars {
			fmt.Printf("%s = %s (type: %s)\n", v.Name, v.Value, v.Type)
		}
	}
}

// handleStep executes a single step forward
func (c *CLI) handleStep() {
	// First step in Delve if available
	if c.debugger != nil {
		fmt.Println("Stepping with Delve...")
		state, err := c.debugger.Step()
		if err != nil {
			fmt.Printf("Delve debugger error: %v\n", err)
		} else if state != nil {
			fmt.Printf("Debugger stepped to: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)

			// Show current variables if available
			c.showCurrentVariables()
		}
	}

	// Then step in the replayer
	currentIdx := c.replayer.CurrentIndex()
	nextIdx := currentIdx + 1
	if err := c.replayer.ReplayToEventIndex(nextIdx); err != nil {
		fmt.Printf("Error stepping forward in replayer: %v\n", err)
		return
	}

	events := c.replayer.Events()
	if nextIdx >= 0 && nextIdx < len(events) {
		fmt.Printf("Stepped to event: %s\n", c.formatEvent(events[nextIdx]))
	}
}

// syncDebuggerToEvent tries to synchronize the debugger state with the current event
func (c *CLI) syncDebuggerToEvent(eventIdx int) error {
	events := c.replayer.Events()
	if eventIdx < 0 || eventIdx >= len(events) {
		return fmt.Errorf("invalid event index: %d", eventIdx)
	}

	event := events[eventIdx]
	fmt.Printf("Synchronizing debugger to event: %s\n", c.formatEvent(event))

	// Try multiple synchronization strategies

	// 1. First, check if the event has precise file and line information
	if event.File != "" && event.Line > 0 {
		// Try setting a breakpoint at this exact location
		bp, err := c.debugger.SetBreakpoint(event.File, event.Line)
		if err == nil {
			fmt.Printf("Set breakpoint at %s:%d for synchronization\n", event.File, event.Line)

			// Continue to this breakpoint
			state, contErr := c.debugger.Continue()

			// Clean up the temporary breakpoint
			err := c.debugger.ClearBreakpoint(bp.ID)
			if err != nil {
				fmt.Printf("Warning: failed to clear temporary breakpoint: %v\n", err)
			}

			if contErr != nil {
				return fmt.Errorf("failed to continue to location %s:%d: %v", event.File, event.Line, contErr)
			}

			if state != nil {
				fmt.Printf("Debugger synchronized, stopped at: %s:%d\n",
					state.CurrentThread.File, state.CurrentThread.Line)
				return nil
			}
		} else {
			fmt.Printf("Could not set breakpoint at %s:%d: %v\n", event.File, event.Line, err)
			// Continue with alternative strategies
		}
	}

	// 2. If event is function entry, try synchronizing to function
	if event.Type == recorder.FuncEntry && event.FuncName != "" {
		fmt.Printf("Attempting to synchronize to function: %s\n", event.FuncName)

		// Use function breakpoint in Delve
		// Note: Requires implementing SetFunctionBreakpoint (not shown here)
		// Use goroutine/function name matching as fallback

		goroutines, err := c.debugger.ListGoroutines()
		if err == nil {
			// Try to find a goroutine currently in this function
			for _, g := range goroutines {
				if g.CurrentLoc.Function != nil &&
					strings.Contains(g.CurrentLoc.Function.Name(), event.FuncName) {
					// Switch to this goroutine
					_, err := c.debugger.client.SwitchGoroutine(g.ID)
					if err == nil {
						fmt.Printf("Switched to goroutine %d at function %s\n",
							g.ID, g.CurrentLoc.Function.Name())
						return nil
					}
				}
			}
		}
	}

	// 3. For variable assignments, try to match variable names and values
	if event.Type == recorder.VarAssignment && event.Details != "" {
		fmt.Printf("Attempting to synchronize based on variable assignment: %s\n", event.Details)
		// This is more complex and would require comparing program state
		// with the recorded variable values
	}

	// 4. If other strategies fail, try finding nearby events with valid file/line info
	searchRadius := 5 // Look 5 events before and after
	for offset := 1; offset <= searchRadius; offset++ {
		// Check events before and after
		for _, checkIdx := range []int{eventIdx - offset, eventIdx + offset} {
			if checkIdx >= 0 && checkIdx < len(events) {
				checkEvent := events[checkIdx]
				if checkEvent.File != "" && checkEvent.Line > 0 {
					fmt.Printf("Trying sync with nearby event at %s:%d\n",
						checkEvent.File, checkEvent.Line)

					// Try setting breakpoint at nearby event
					nearbyBp, err := c.debugger.SetBreakpoint(checkEvent.File, checkEvent.Line)
					if err == nil {
						// Continue to this nearby point
						state, contErr := c.debugger.Continue()

						// Clean up temporary breakpoint
						clearErr := c.debugger.ClearBreakpoint(nearbyBp.ID)
						if clearErr != nil {
							fmt.Printf("Warning: failed to clear temporary breakpoint: %v\n", clearErr)
						}

						if contErr == nil && state != nil {
							fmt.Printf("Synchronized to nearby location: %s:%d\n",
								state.CurrentThread.File, state.CurrentThread.Line)

							// If we hit a point after our target, we may need to step backward
							// If we hit a point before our target, we may need to step forward
							if checkIdx < eventIdx {
								// Need to step forward to reach target
								fmt.Println("Need to step forward to reach exact event")
							} else {
								// We're past the target event
								fmt.Println("Synchronized beyond target event")
							}
							return nil
						}
					}
				}
			}
		}
	}

	// 5. Last resort: Reset program state and try a more holistic approach
	fmt.Println("Precise synchronization failed, attempting approximate sync...")

	// Get current state for reference
	state, err := c.debugger.client.GetState()
	if err == nil && state.CurrentThread != nil {
		fmt.Printf("Current debugger position: %s:%d\n",
			state.CurrentThread.File, state.CurrentThread.Line)
	}

	return fmt.Errorf("could not precisely synchronize debugger state to event %d", eventIdx)
}

// resetDebuggerToEvent restarts the Delve debugger and brings it to a state matching the current event
func (c *CLI) resetDebuggerToEvent(eventIdx int) error {
	if c.debugger == nil {
		return nil // No debugger to reset
	}

	events := c.replayer.Events()
	if eventIdx < 0 || eventIdx >= len(events) {
		return fmt.Errorf("invalid event index: %d", eventIdx)
	}

	fmt.Println("Resetting debugger state to match replayer...")

	// Get the current target program
	targetPath := c.debugger.target

	// Store existing breakpoints before closing so we can restore them
	var existingBreakpoints []*api.Breakpoint
	if c.debugger.client != nil {
		bps, err := c.debugger.client.ListBreakpoints(false)
		if err == nil {
			existingBreakpoints = bps
		}
	}

	// Close the current debugger session
	if err := c.debugger.Close(); err != nil {
		fmt.Printf("Warning: error closing debugger: %v\n", err)
	}

	// Create a new debugger session
	var err error
	c.debugger, err = NewDelveDebugger(targetPath)
	if err != nil {
		return fmt.Errorf("failed to restart debugger: %v", err)
	}

	// Restore previous breakpoints
	for _, bp := range existingBreakpoints {
		// Skip internal breakpoints (those with special IDs)
		if bp.ID <= 0 {
			continue
		}

		// Restore based on file:line
		if bp.File != "" && bp.Line > 0 {
			fmt.Printf("Restoring breakpoint at %s:%d\n", bp.File, bp.Line)
			_, err := c.debugger.SetBreakpoint(bp.File, bp.Line)
			if err != nil {
				fmt.Printf("Warning: failed to restore breakpoint at %s:%d: %v\n",
					bp.File, bp.Line, err)
			}
		}
	}

	// Build a map of all available file:line locations from recorded events
	// This helps with finding the nearest valid execution point
	validLocations := make(map[string]bool)
	for _, e := range events {
		if e.File != "" && e.Line > 0 {
			key := fmt.Sprintf("%s:%d", e.File, e.Line)
			validLocations[key] = true
		}
	}

	// Try setting a breakpoint at main() to start
	mainBp, _ := c.debugger.client.CreateBreakpoint(&api.Breakpoint{
		FunctionName: "main.main",
	})

	if mainBp != nil {
		// Continue to main
		_, _ = c.debugger.Continue()
		// Clean up main breakpoint
		clearErr := c.debugger.ClearBreakpoint(mainBp.ID)
		if clearErr != nil {
			fmt.Printf("Warning: failed to clear main breakpoint: %v\n", clearErr)
		}
	}

	// Now try to synchronize to the specific event
	err = c.syncDebuggerToEvent(eventIdx)
	if err != nil {
		// If sync fails, try to get reasonably close
		fmt.Printf("Warning: precise sync failed, using best-effort approach: %v\n", err)

		// Look for events leading up to current one with valid locations
		var validEvents []recorder.Event
		for i := 0; i <= eventIdx; i++ {
			if events[i].File != "" && events[i].Line > 0 {
				validEvents = append(validEvents, events[i])
			}
		}

		// Try setting breakpoints at several valid locations
		// preceding our target and continue to each
		for i := len(validEvents) - 1; i >= 0 && i >= len(validEvents)-5; i-- {
			e := validEvents[i]
			bp, err := c.debugger.SetBreakpoint(e.File, e.Line)
			if err == nil {
				// Try continuing to this point
				if state, err := c.debugger.Continue(); err == nil && state != nil {
					fmt.Printf("Reset to approximate position: %s:%d\n",
						state.CurrentThread.File, state.CurrentThread.Line)
					clearErr := c.debugger.ClearBreakpoint(bp.ID)
					if clearErr != nil {
						fmt.Printf("Warning: failed to clear breakpoint: %v\n", clearErr)
					}
					break
				}
				clearErr := c.debugger.ClearBreakpoint(bp.ID)
				if clearErr != nil {
					fmt.Printf("Warning: failed to clear breakpoint: %v\n", clearErr)
				}
			}
		}
	}

	return nil
}

// handleBackstep steps backward one event
func (c *CLI) handleBackstep() {
	currentIdx := c.replayer.CurrentIndex()
	newIdx, err := c.replayer.StepBackward(currentIdx)
	if err != nil {
		fmt.Printf("Error stepping backward: %v\n", err)
		return
	}

	events := c.replayer.Events()
	if newIdx >= 0 && newIdx < len(events) {
		fmt.Printf("Stepped back to event: %s\n", c.formatEvent(events[newIdx]))

		// If Delve is available, reset the debugging session
		// to match the replayer's new state, as Delve can't step backward
		if c.debugger != nil {
			if err := c.resetDebuggerToEvent(newIdx); err != nil {
				fmt.Printf("Error synchronizing debugger state: %v\n", err)
			}
		}
	}
}

// handleInfo shows current execution state
func (c *CLI) handleInfo() {
	events := c.replayer.Events()
	idx := c.replayer.CurrentIndex()
	if idx >= 0 && idx < len(events) {
		fmt.Printf("\nCurrent event: %s\n", c.formatEvent(events[idx]))
	} else {
		fmt.Println("No current event")
	}

	// If Delve is available, show debugger state
	if c.debugger != nil {
		state, err := c.debugger.client.GetState()
		if err != nil {
			fmt.Printf("Error getting debugger state: %v\n", err)
			return
		}

		if state.CurrentThread != nil {
			fmt.Printf("\nDebugger state:\n")
			fmt.Printf("  Current position: %s:%d\n", state.CurrentThread.File, state.CurrentThread.Line)
			if state.CurrentThread.Function != nil {
				fmt.Printf("  Current function: %s\n", state.CurrentThread.Function.Name())
			}
			fmt.Printf("  Goroutine: %d\n", state.CurrentThread.GoroutineID)
		}
	}
}

// Delve-specific command handlers

// handleBreakpoint sets a breakpoint at the specified location
func (c *CLI) handleBreakpoint(args []string) {
	if c.debugger == nil {
		fmt.Println("Delve integration not enabled")
		return
	}

	if len(args) < 1 {
		fmt.Println("Usage: breakpoint <file:line> or func:<function_name>")
		fmt.Println("Optional: breakpoint <file:line> -c <condition>")
		return
	}

	// Check for conditional breakpoint syntax
	var condition string
	var locationArg string

	if len(args) >= 3 && args[1] == "-c" {
		// Format: breakpoint file:line -c condition
		locationArg = args[0]
		condition = args[2]
	} else {
		// Standard breakpoint
		locationArg = args[0]
	}

	// Check if this is a function breakpoint
	if strings.HasPrefix(locationArg, "func:") {
		funcName := strings.TrimPrefix(locationArg, "func:")

		// Set a function breakpoint
		dbp, err := c.debugger.SetFunctionBreakpoint(funcName)
		if err != nil {
			fmt.Printf("Error setting function breakpoint: %v\n", err)
			return
		}

		// Add to our breakpoint manager
		bp, err := c.bpManager.AddBreakpoint("func:" + funcName)
		if err != nil {
			fmt.Printf("Warning: Error adding breakpoint to manager: %v\n", err)
		}

		fmt.Printf("Function breakpoint %d set at %s (Delve bp: %d)\n",
			bp.ID, funcName, dbp.ID)
		return
	}

	// Parse file:line format with special handling for Windows paths
	input := locationArg

	// Convert any backslashes to forward slashes for consistency
	input = strings.ReplaceAll(input, "\\", "/")

	// Find the last colon, which should separate the file path from line number
	lastColonIndex := strings.LastIndex(input, ":")
	if lastColonIndex == -1 {
		fmt.Println("Invalid format. Use file:line (e.g., main.go:42) or func:functionName")
		return
	}

	file := input[:lastColonIndex]
	lineStr := input[lastColonIndex+1:]

	// Parse line number
	line, err := strconv.Atoi(lineStr)
	if err != nil {
		fmt.Printf("Invalid line number: %v\n", err)
		return
	}

	// Set breakpoint in the Delve debugger
	var dbp *api.Breakpoint
	var breakpointErr error

	if condition != "" {
		// Set conditional breakpoint
		dbp, breakpointErr = c.debugger.SetConditionalBreakpoint(file, line, condition)
	} else {
		// Regular breakpoint
		dbp, breakpointErr = c.debugger.SetBreakpoint(file, line)
	}

	if breakpointErr != nil {
		fmt.Printf("Error setting breakpoint: %v\n", breakpointErr)
		return
	}

	// Also add the breakpoint to our own manager
	bp, err := c.bpManager.AddBreakpoint(fmt.Sprintf("%s:%d", file, line))
	if err != nil {
		fmt.Printf("Warning: Error adding breakpoint to manager: %v\n", err)
	}

	if condition != "" {
		fmt.Printf("Conditional breakpoint %d set at %s:%d with condition '%s' (Delve bp: %d)\n",
			bp.ID, file, line, condition, dbp.ID)
	} else {
		fmt.Printf("Breakpoint %d set at %s:%d (Delve bp: %d)\n", bp.ID, file, line, dbp.ID)
	}
}

// handleListBreakpoints lists all breakpoints
func (c *CLI) handleListBreakpoints() {
	fmt.Println("\nBreakpoints:")

	// Show our managed breakpoints
	for _, bp := range c.GetBreakpoints() {
		status := "enabled"
		if !bp.Enabled {
			status = "disabled"
		}

		switch bp.Type {
		case LocationBreakpoint:
			fmt.Printf("%d: %s:%d (location) [%s]\n", bp.ID, bp.File, bp.Line, status)
		case FunctionBreakpoint:
			fmt.Printf("%d: %s (function) [%s]\n", bp.ID, bp.Function, status)
		case EventTypeBreakpoint:
			fmt.Printf("%d: %s (event) [%s]\n", bp.ID, bp.EventType, status)
		}
	}

	// If Delve is enabled, also show Delve breakpoints
	if c.debugger != nil {
		// Get breakpoints from Delve API
		breakpoints, err := c.debugger.client.ListBreakpoints(false)
		if err != nil {
			fmt.Printf("Error listing Delve breakpoints: %v\n", err)
			return
		}

		// Show Delve breakpoints if any
		if len(breakpoints) > 0 {
			fmt.Println("\nDelve Breakpoints:")
			for _, bp := range breakpoints {
				status := "enabled"
				if bp.Disabled {
					status = "disabled"
				}

				fmt.Printf("[%d] %s:%d %s (%s)\n",
					bp.ID, bp.File, bp.Line, bp.FunctionName, status)
			}
		}
	}
}

// handlePrintVariable prints the value of a variable
func (c *CLI) handlePrintVariable(args []string) {
	if c.debugger == nil {
		fmt.Println("Delve integration not enabled")
		return
	}

	if len(args) < 1 {
		fmt.Println("Usage: print <variable>")
		return
	}

	varName := args[0]
	v, err := c.debugger.GetVariable(varName)
	if err != nil {
		fmt.Printf("Error getting variable '%s': %v\n", varName, err)
		return
	}

	fmt.Printf("%s = %s (type: %s)\n", v.Name, v.Value, v.Type)
}

// handleListGoroutines lists all goroutines
func (c *CLI) handleListGoroutines() {
	if c.debugger == nil {
		fmt.Println("Delve integration not enabled")
		return
	}

	goroutines, err := c.debugger.ListGoroutines()
	if err != nil {
		fmt.Printf("Error listing goroutines: %v\n", err)
		return
	}

	fmt.Printf("Found %d goroutines:\n", len(goroutines))
	for i, g := range goroutines {
		fmt.Printf("[%d] Goroutine %d", i, g.ID)
		if g.CurrentLoc.Function != nil {
			fmt.Printf(" - %s (%s:%d)", g.CurrentLoc.Function.Name(), g.CurrentLoc.File, g.CurrentLoc.Line)
		}
		fmt.Println()
	}
}

// handleWatch handles the watch command
func (c *CLI) handleWatch(args []string) {
	if len(args) < 1 {
		fmt.Println("Usage: watch [-r|-w|-rw] <expression>")
		fmt.Println("  -r    stops when the memory location is read")
		fmt.Println("  -w    stops when the memory location is written")
		fmt.Println("  -rw   stops when the memory location is read or written (default)")
		return
	}

	// Parse flags
	var readFlag, writeFlag bool = true, true // Default to -rw
	var expr string

	if args[0] == "-r" {
		if len(args) < 2 {
			fmt.Println("Expression required")
			return
		}
		readFlag, writeFlag = true, false
		expr = args[1]
	} else if args[0] == "-w" {
		if len(args) < 2 {
			fmt.Println("Expression required")
			return
		}
		readFlag, writeFlag = false, true
		expr = args[1]
	} else if args[0] == "-rw" {
		if len(args) < 2 {
			fmt.Println("Expression required")
			return
		}
		readFlag, writeFlag = true, true
		expr = args[1]
	} else {
		// No flag, use the first arg as expression with default -rw
		expr = args[0]
	}

	// Determine watchpoint type for our manager
	var watchType BreakpointType
	if readFlag && writeFlag {
		watchType = WatchpointReadWrite
	} else if readFlag {
		watchType = WatchpointRead
	} else {
		watchType = WatchpointWrite
	}

	// Try to set watchpoint in Delve if it's active
	var watchDbp *api.Breakpoint
	var delveErr error

	if c.debugger != nil {
		watchDbp, delveErr = c.debugger.SetWatchpoint(expr, readFlag, writeFlag)
		if delveErr != nil {
			// Don't return error immediately - we'll still create a replay watchpoint
			fmt.Printf("Warning: Unable to set live Delve watchpoint: %v\n", delveErr)
			fmt.Println("Creating replay-only watchpoint instead.")
		}
	} else {
		fmt.Println("Delve integration not active. Creating replay-only watchpoint.")
	}

	// Add to our breakpoint manager (for replay mode)
	watchBp, err := c.bpManager.AddWatchpoint(expr, watchType)
	if err != nil {
		fmt.Printf("Error adding watchpoint to manager: %v\n", err)
		return
	}

	if watchDbp != nil {
		fmt.Printf("Watchpoint %d set on expression '%s' (Delve bp: %d)\n",
			watchBp.ID, expr, watchDbp.ID)
	} else {
		fmt.Printf("Replay watchpoint %d set on expression '%s'\n",
			watchBp.ID, expr)
		fmt.Println("Note: This watchpoint will work during event replay only.")
		fmt.Println("      Look for variable changes in recorded events.")
	}
}

// GetBreakpoints returns all breakpoints
func (c *CLI) GetBreakpoints() []*Breakpoint {
	return c.bpManager.GetBreakpoints()
}
