//go:build windows

package main

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// setHideWindowForSubprocess stops console windows from flashing when a GUI .exe
// spawns short-lived command-line tools (e.g. netsh).
func setHideWindowForSubprocess(cmd *exec.Cmd) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	cmd.SysProcAttr.HideWindow = true
	cmd.SysProcAttr.CreationFlags = windows.CREATE_NO_WINDOW
}
