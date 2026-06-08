# Manitor

Manitor is a lightweight network-usage monitoring system made up of three services:

| Service | Role | Where it runs |
|---------|------|---------------|
| **Client** | Collects upload/download traffic on each PC and sends it to the server | Every monitored Windows machine |
| **Server** | Receives telemetry, stores it in PostgreSQL, and exposes a REST/WebSocket API | Central host reachable by all clients |
| **Dashboard** | Web UI for system managers to browse connections and live traffic charts | Separate host (or same machine as the server) |

```
  ┌─────────────┐     POST /api/v1/connections      ┌─────────────┐
  │   Client    │ ─────────────────────────────────►│   Server    │
  │  (Windows)  │                                   │   (Go API)  │
  └─────────────┘                                   └──────┬──────┘
                                                           │
                                                           ▼
                                                    ┌─────────────┐
                                                    │ PostgreSQL  │
                                                    └─────────────┘
                                                           ▲
                                                           │
  ┌─────────────┐     GET /api/v1/connections       ┌──────┴──────┐
  │  Dashboard  │ ◄───────────────────────────────│   Server    │
  │  (React UI) │     WebSocket stream              └─────────────┘
  └─────────────┘
```

---

## Prerequisites

- **PostgreSQL** — required by the server (create an empty database before first run)
- **Go 1.22+** — to build/run the client and server
- **Node.js 22+** and **pnpm** — to build/run the dashboard
- **Windows** — the client agent is Windows-only (system tray, `netsh` Wi‑Fi detection)

---

## Quick start (local development)

### 1. Database

Create a PostgreSQL database (example name: `manitor`), then run migrations:

```powershell
cd server
copy .env.sample .env
# Edit .env with your DB_* values

go run ./cmd/migrate
```

### 2. Server

```powershell
cd server
go mod tidy
go run .
```

The API listens on the address in `SERVER_CLIENT` (use `:5000` for local dev). Verify with:

```text
GET http://localhost:5000/health
```

### 3. Dashboard

```powershell
cd dashboard
copy .env.sample .env
```

Set in `.env`:

```env
VITE_SERVER_BASE_URL=http://localhost:5000
```

```powershell
pnpm install
pnpm dev
```

Open the URL Vite prints (usually `http://localhost:5173`). In dev mode, Vite proxies `/api` requests to the server.

### 4. Client

```powershell
cd client
copy .env.sample .env
```

Set in `.env` (place this file **next to the built `.exe`** on each machine):

```env
SERVER_URL=http://localhost:5000/api/v1/connections
```

```powershell
go mod tidy
go run .
```

For a release build with no console window:

```powershell
go build -ldflags "-H=windowsgui" -o manitor-client.exe .
```

---

## Client (Windows agent)

Install the built executable on every PC you want to monitor.

### Setup

1. Copy `manitor-client.exe` to the target machine.
2. Create a `.env` file in the **same folder** as the executable:

   ```env
   SERVER_URL=http://your-server-host:5000/api/v1/connections
   ```

   Use the full ingest URL, including `/api/v1/connections`. The client reads this at startup from `.env` in the working directory.

3. Double-click `manitor-client.exe`. A tray icon appears in the Windows notification area (no console window in release builds).

### Tray menu

| Action | Description |
|--------|-------------|
| **Activate** | Start collecting and sending traffic every second. While active, the label changes to **Deactivate**. |
| **Deactivate** | Stop monitoring without quitting the app. |
| **Check logs** | Opens a PowerShell window that tails the client log file. |
| **Exit** | Stops monitoring and closes the application. |

### Logs

Logs are written to:

```text
%APPDATA%\Manitor\client.log
```

For debugging with a visible console:

```powershell
$env:MANITOR_DEBUG_CONSOLE = "1"
go run .
```

### What is collected

Each second (while activated), the client sends:

- Machine display name and local IP
- Connected Wi‑Fi name (SSID)
- Per-interval upload and download byte totals across active network interfaces

---

## Server (API + storage)

Host the server somewhere all clients can reach over HTTP. It persists data in **PostgreSQL** via GORM.

### Environment (`.env`)

| Variable | Description | Example |
|----------|-------------|---------|
| `SERVER_CLIENT` | Address the HTTP server binds to | `:5000` |
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | `postgres` |
| `DB_PASSWORD` | Database password | `secret` |
| `DB_NAME` | Database name | `manitor` |
| `DB_SSL_MODE` | PostgreSQL SSL mode | `disable` |

### First-time setup

```powershell
cd server
go run ./cmd/migrate   # creates/updates the connections table
go run .               # start the API
```

### API overview

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` | Health check |
| `POST` | `/api/v1/connections` | Ingest client telemetry |
| `GET` | `/api/v1/connections` | List connections (supports sorting/pagination) |
| `GET` | `/api/v1/connections/stream` | WebSocket live updates for a host + Wi‑Fi session |

**List query examples:**

```text
/api/v1/connections?total_download=desc&limit=100
/api/v1/connections?total_upload=asc
```

**WebSocket example:**

```text
ws://your-server:5000/api/v1/connections/stream?host_name=Jane&wifi_name=HomeNet
```

### Docker

```powershell
docker build -t manitor-server ./server
```

Run with the same environment variables passed in (see `server/.env.sample`). The image listens on port `5000` inside the container.

A `docker-compose.yml` at the repo root can run the server and dashboard together; adjust `DB_*` and `VITE_SERVER_BASE_URL` for your environment. The compose file expects an external Docker network named `devopsio` and a reachable PostgreSQL instance.

---

## Dashboard (web UI)

Host the dashboard where system managers can open it in a browser. It connects to the **server API** (not directly to PostgreSQL).

### Environment

| Variable | Description | Example |
|----------|-------------|---------|
| `VITE_SERVER_BASE_URL` | Public base URL of the server API (no trailing slash) | `https://api.example.com` |

This value is baked in at **build time** for production builds.

### Development

```powershell
cd dashboard
pnpm install
# .env with VITE_SERVER_BASE_URL=http://localhost:5000
pnpm dev
```

### Production build

```powershell
cd dashboard
pnpm install
$env:VITE_SERVER_BASE_URL = "https://your-server.example.com"
pnpm build
```

Serve the `dist/` folder with any static file host (nginx, Caddy, S3 + CDN, etc.).

### Docker

```powershell
docker build -t manitor-dashboard ./dashboard --build-arg VITE_SERVER_BASE_URL=http://your-server:5000
```

The image serves the built app on port `80` via nginx.

### Features

- Table of all connections: IP, display name, Wi‑Fi, cumulative upload/download
- Live traffic chart (WebSocket) per display name + Wi‑Fi pair

**CORS:** The server allows `http://localhost:5173` and `http://localhost:3000` by default. If you host the dashboard on another origin, add that origin in `server/config/cors.go` and rebuild the server.

---

## Deployment checklist

Use this when rolling out to a real environment:

### Server + database

- [ ] Provision PostgreSQL and create the `manitor` database
- [ ] Copy `server/.env.sample` → `server/.env` and fill in all `DB_*` and `SERVER_CLIENT` values
- [ ] Run `go run ./cmd/migrate` once
- [ ] Start the server (binary, systemd, or Docker)
- [ ] Confirm `GET /health` responds from client machines and the dashboard host
- [ ] Open firewall port for the API (default `5000`)

### Dashboard

- [ ] Set `VITE_SERVER_BASE_URL` to the **browser-reachable** server URL (not an internal Docker hostname unless managers use a VPN)
- [ ] Build and deploy `dashboard/dist` or the dashboard Docker image
- [ ] Update server CORS if the dashboard origin is not localhost

### Clients (each Windows PC)

- [ ] Copy `manitor-client.exe` and `.env` to the same directory
- [ ] Set `SERVER_URL` to `http(s)://<server>/api/v1/connections`
- [ ] Run the executable; use **Activate** to begin monitoring

---

## Repository layout

```text
manitor/
├── client/       # Windows tray agent (Go)
├── server/       # HTTP API + PostgreSQL (Go)
├── dashboard/    # React + Vite admin UI
└── docker-compose.yml
```

More detail per component:

- [client/README.md](client/README.md) — client behavior and build flags
- [server/README.md](server/README.md) — API reference notes

---

## License

See [LICENSE](LICENSE).
