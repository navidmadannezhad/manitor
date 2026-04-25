# Manitor Server Service

Go HTTP service that receives client telemetry and stores normalized connection rows in SQLite.

## Environment

- Uses `.env` with `SERVER_CLIENT` for server bind target.
- You can set either `:5000` or `http://localhost:5000`.
- Default fallback: `:5000`.

## Features

- Ingest endpoint at `POST /api/v1/connections` (also accepts `POST /`)
- SQLite storage (`manitor.db`) with `connections` table
- Incremental total fields per IP:
  - `total_download = previous_total_download + download_size`
  - `total_upload = previous_total_upload + upload_size`
- Daily reset at local midnight (`00:00`) that fully clears data
- Health endpoint at `GET /health`
- Listing endpoint at `GET /api/v1/connections` with sorting query support
- Realtime socket endpoint at `GET /api/v1/connections/:ip` (WebSocket)

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

- `ws://localhost:5000/api/v1/connections/172.16.0.2`

Server messages:

- `type=history` -> full chart history for the IP
- `type=update` -> newly inserted rows every second

## Connection model

- `id` (primary key)
- `ip`
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
