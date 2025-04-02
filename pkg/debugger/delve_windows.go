//go:build windows
// +build windows

package debugger

import (
	"os/exec"
	"syscall"
)

// setupProcAttr configures platform-specific process attributes.
// On Windows, this prevents Delve from creating a console window.
func setupProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}
