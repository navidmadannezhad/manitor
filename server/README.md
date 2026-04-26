# Manitor Server Service

Go HTTP service that receives client telemetry and stores normalized connection rows in SQLite.

## Environment

- Uses `.env` with `SERVER_CLIENT` for server bind target.
- You can set either `:5000` or `http://localhost:5000`.
- Default fallback: `:5000`.

## Features

- Ingest endpoint at `POST /api/v1/connections` (also accepts `POST /`)
- SQLite storage (`manitor.db`) with `connections` table
- Incremental total fields per **display name + Wi‑Fi** pair (same `hostname` and `wifiname` as the previous row for that pair):
  - `total_download = previous_total_download + download_size`
  - `total_upload = previous_total_upload + upload_size`
- Daily reset at local midnight (`00:00`) that fully clears data
- Health endpoint at `GET /health`
- Listing endpoint at `GET /api/v1/connections` with sorting query support
- Realtime socket at `GET /api/v1/connections/stream?host_name=...&wifi_name=...` (WebSocket; query values are normalized like ingest)

## Query sorting for list API

`GET /api/v1/connections` supports:

- `total_download=asc|desc` (primary sort)
- `total_upload=asc|desc` (secondary sort)
- `sort_by=total_download|total_upload` with `order=asc|desc` (fallback option)
- `limit=<n>` (default `500`, max `5000`)

Examples:

- `/api/v1/connections?total_download=desc`
- `/api/v1/connections?total_download=desc&total_upload=asc&limit=1000`

## Realtime socket API

Connect to:

- `ws://localhost:5000/api/v1/connections/stream?host_name=Jane&wifi_name=HomeNet` (URL-encode as needed)

Server messages:

- `type=history` -> full chart history for that display name + Wi‑Fi session
- `type=update` -> newly inserted rows for that session when present (otherwise connection stays open with no message; chart can stay static)

## Connection model

- `id` (primary key)
- `ip` (agent system IP; informational)
- `hostname` (display name from agent `host_name`)
- `wifiname`
- `download_size`
- `upload_size`
- `total_download`
- `total_upload`
- `created_at`

## Run

```powershell
go mod tidy
go run .
```

Server starts on `:5000`.
