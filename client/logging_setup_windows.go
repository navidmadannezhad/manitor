//go:build windows

package main

import (
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
)

const logFileName = "client.log"
const appConfigDirName = "Manitor"
const envDebugConsole = "MANITOR_DEBUG_CONSOLE"

var logFilePath string

// initLoggingForWindows should run at the start of main() before any log output.
// By default it detaches from the process console so double-clicking does not show a window.
// Set MANITOR_DEBUG_CONSOLE=1 to keep a console (e.g. go run, manual testing).
func initLoggingForWindows() {
	if os.Getenv(envDebugConsole) == "1" {
		if w := openOrCreateLogFile(); w != nil {
			log.SetOutput(io.MultiWriter(os.Stderr, w))
		}
		return
	}
	// Close the auto-allocated console when the EXE is a console subsystem (normal double-click / shortcut).
	freeConsole()
	attachFileLogger()
}

func freeConsole() {
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	p := k32.NewProc("FreeConsole")
	_, _, _ = p.Call()
}

func openOrCreateLogFile() *os.File {
	p, err := clientLogFilePath()
	if err != nil {
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil
	}
	logFilePath = p
	return f
}

func attachFileLogger() {
	f := openOrCreateLogFile()
	if f == nil {
		return
	}
	log.SetOutput(f)
}

func clientLogFilePath() (string, error) {
	if logFilePath != "" {
		return logFilePath, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, appConfigDirName, logFileName), nil
}

// openLogViewerInTerminal spawns a new window running PowerShell that tails the log file.
func openLogViewerInTerminal() {
	p, err := clientLogFilePath()
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(p), 0o700)
	if f, e := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600); e == nil {
		_ = f.Close()
	}
	logFilePath = p
	ps := `$p=$env:MANITOR_LOG_PATH; if (-not (Test-Path -LiteralPath $p)) { New-Item -ItemType File -Path $p -Force | Out-Null }; Get-Content -LiteralPath $p -Wait -Tail 100`
	cmd := exec.Command("cmd", "/c", "start", "Manitor logs", "powershell", "-NoExit", "-NoLogo", "-NoProfile", "-Command", ps)
	cmd.Env = append(os.Environ(), "MANITOR_LOG_PATH="+p)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: false}
	_ = cmd.Start()
}
