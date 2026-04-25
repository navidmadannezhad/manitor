# Manitor Client Agent

Windows tray client agent that collects lightweight network telemetry every second and sends it to `http://localhost:5000/api/v1/connections`.

## Environment

- Uses `.env` with `SERVER_CLIENT` to choose ingest API URL.
- Default fallback: `http://localhost:5000/api/v1/connections`

## Behavior

- Stays idle until user clicks **Activate** from the tray menu.
- Once activated:
  - Prints realtime debug logs to terminal every second (`[realtime] ...`).
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
