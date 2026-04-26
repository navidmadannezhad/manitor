package main

import (
	"os"
	"os/user"
	"strings"
	"sync"
)

var (
	agentFriendlyNameOnce sync.Once
	agentFriendlyNameVal  string
)

// agentFriendlyName is sent as host_name in the JSON payload (DB column hostname).
// Windows: prefers .NET UserPrincipal.DisplayName (Microsoft account / local “full name”).
// Linux/macOS and others: prefers passwd GECOS via user.Current().Name, then login, then hostname.
// Resolved once per process so we do not run PowerShell on every telemetry tick.
func agentFriendlyName() string {
	agentFriendlyNameOnce.Do(func() {
		agentFriendlyNameVal = strings.TrimSpace(resolveAgentFriendlyName())
		if agentFriendlyNameVal == "" {
			agentFriendlyNameVal = "unknown"
		}
	})
	return agentFriendlyNameVal
}

func resolveAgentFriendlyName() string {
	if d := windowsAccountDisplayName(); d != "" {
		return d
	}
	u, err := user.Current()
	if err == nil {
		if n := strings.TrimSpace(u.Name); n != "" {
			return n
		}
		if n := strings.TrimSpace(u.Username); n != "" {
			return n
		}
	}
	if h, err := os.Hostname(); err == nil {
		return strings.TrimSpace(h)
	}
	return ""
}
