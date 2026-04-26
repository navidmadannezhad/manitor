//go:build !windows

package main

import "os/exec"

func setHideWindowForSubprocess(_ *exec.Cmd) {}
