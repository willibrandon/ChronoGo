//go:build !windows
// +build !windows

package debugger

import (
	"os/exec"
)

// setupProcAttr configures platform-specific process attributes.
// On non-Windows platforms, this is a no-op.
func setupProcAttr(cmd *exec.Cmd) {
	// No special attributes needed for Unix-like systems
}
