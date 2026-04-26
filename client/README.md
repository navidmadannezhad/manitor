# Manitor Client Agent

Windows tray client agent that collects lightweight network telemetry every second and sends it to `http://localhost:5000/api/v1/connections`.

## Server URL

- The ingest URL is the constant **`serverClientURL`** at the top of `main.go` (change it and rebuild). The client does not read `SERVER_CLIENT` or `.env` for this.

## Behavior

- Stays idle until the user chooses **Activate** from the tray menu; while running, the same item reads **Deactivate** and stops telemetry without quitting the app.
- No console window on startup. Logs are written to `%APPDATA%\Manitor\client.log` (after `os.UserConfigDir()` + `Manitor\client.log`). Use **Check logs** in the tray menu to open a **new** PowerShell window that tails that file.
- Set `MANITOR_DEBUG_CONSOLE=1` to keep the parent console (for `go run` / debugging); otherwise the app detaches from the auto-opened console so double-clicking does not leave a terminal open.
- Once activated:
  - Writes realtime debug lines to the log file every second (`[realtime] ...`).
  - Collects telemetry immediately, then every second.
  - Sends payload with a request timeout of 30 seconds.
  - Retries sending up to 3 times on failure.
  - If the 3rd attempt fails, that interval is dropped and the agent waits for the next cycle.
- Clicking **Exit** deactivates the agent and closes the app.

## Collected telemetry fields

- `system_ip`
- `wifi_name`
- `logs[].request_url`
  - Currently set to `system://all-interfaces`.
- `logs[].packet_size`
  - Aggregate upload/download byte deltas across active interfaces for the collection window.
- `logs[].direction`
  - `upload` or `download`

## Run

```powershell
go mod tidy
go run .
```

## Build

```powershell
go build -o manitor-client.exe .
```

For a release build with **no** console subsystem (no flash, no extra `FreeConsole` dependency on behavior), use:

```powershell
go build -ldflags "-H=windowsgui" -o manitor-client.exe .
```
