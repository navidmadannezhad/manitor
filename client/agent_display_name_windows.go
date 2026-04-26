//go:build windows

package main

import (
	"context"
	"os/exec"
	"strings"
	"time"
)

func windowsAccountDisplayName() string {
	// DisplayName matches what many users expect from a Microsoft account / local full name.
	ps := `Add-Type -AssemblyName System.DirectoryServices.AccountManagement; try { $u = [DirectoryServices.AccountManagement.UserPrincipal]::Current; if ($null -ne $u -and $u.DisplayName) { $u.DisplayName } } catch { '' }`
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "powershell.exe", "-NoProfile", "-NonInteractive", "-Command", ps)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
