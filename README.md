# Go + Next Roster Dashboard

Go backend plus Next.js frontend for managing monthly employee shift rosters.

The backend is a Go JSON API backed by SQLite. The frontend is a Next.js app ported from the original React UI.

## Prerequisites

- Go 1.22+
- Node.js 18+ or 20+
- npm

## Project Layout

```text
roster_dashboard_go/
├── cmd/server/          # Go API entrypoint
├── internal/            # Go app, auth, DB, and API handlers
├── frontend/            # Next.js frontend
├── go.mod
└── README.md
```

## Quick Start

Clone or copy the project, then run these commands from the project root.

### 1. Start The Backend

```bash
SESSION_SECRET=dev-secret-change-me \
PORT=3001 \
go run ./cmd/server
```

By default, this creates or uses `./roster.db` in the project root.

To reuse an existing roster database, point `DB_PATH` at it:

```bash
DB_PATH=/path/to/existing/roster.db \
SESSION_SECRET=dev-secret-change-me \
PORT=3001 \
go run ./cmd/server
```

The backend creates missing tables and seeds the default admin only when no `admin` user exists.

Default credentials:

| Username | Password |
|---|---|
| `admin` | `admin123` |

### 2. Start The Frontend

Open a second terminal:

```bash
cd frontend
npm install
GO_API_BASE=http://localhost:3001 npm run dev
```

Open:

```text
http://localhost:3000
```

Services:

| Service | URL |
|---|---|
| Next.js frontend | `http://localhost:3000` |
| Go API backend | `http://localhost:3001/api` |

The frontend calls `/api/*`. In development, Next.js rewrites those requests to `GO_API_BASE`.

## Verify Setup

Run backend tests:

```bash
go test ./...
```

Build the frontend:

```bash
cd frontend
npm run build
```

Quick API login check, while the backend is running:

```bash
curl -i -X POST http://localhost:3001/api/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}'
```

## Production

Build the backend:

```bash
go build -o bin/roster-api ./cmd/server
```

Build the frontend:

```bash
cd frontend
npm run build
```

Run the backend:

```bash
APP_ENV=production \
DB_PATH=/path/to/roster.db \
SESSION_SECRET=<strong-random-secret> \
PORT=3001 \
./bin/roster-api
```

Run the frontend:

```bash
cd frontend
GO_API_BASE=http://localhost:3001 npm start
```

In production, run both processes behind a reverse proxy or process manager. Route user traffic to the Next.js app and make sure `/api/*` reaches the Go backend.

## Environment Variables

| Variable | Default | Used by | Description |
|---|---|---|---|
| `PORT` | `3001` | Backend | HTTP listen port |
| `DB_PATH` | `./roster.db` | Backend | SQLite database path |
| `SESSION_SECRET` | development fallback | Backend | HS256 token/session signing secret |
| `JWT_SECRET` | optional | Backend | Used if `SESSION_SECRET` is unset |
| `APP_ENV` | `development` | Backend | Set to `production` to require a real secret |
| `GO_API_BASE` | `http://localhost:3001` | Frontend | Backend URL used by Next.js rewrites |

## Notes

- API routes are exposed under `/api`.
- `/api/auth/login` returns the Bearer token and user payload expected by the frontend.
- The Next.js UI includes the dashboard, roster grid, admin panel, settings, dark mode, and CSV import workflows.
- SQLite is a single-writer database; keep backend deployment to one process unless you intentionally design around that limit.
