package debugger

// ResetDebuggerToEventForTesting is a helper function to expose the resetDebuggerToEvent
// method for testing purposes only.
func ResetDebuggerToEventForTesting(cli *CLI, eventIdx int) error {
	return cli.resetDebuggerToEvent(eventIdx)
}
