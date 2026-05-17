# Go + Next Roster Dashboard

Go backend plus Next.js frontend for managing monthly employee shift rosters.

The backend is a Go JSON API backed by SQLite. The frontend is a Next.js app ported from the original React UI.

## Prerequisites

- Go 1.26.3+
- Node.js 20.9+
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

The backend creates missing tables and seeds the first admin only when no admin user exists.

Development-only default credentials:

| Username | Password |
|---|---|
| `admin` | `admin123` |

In production, the backend will not use the default password. Set `INITIAL_ADMIN_PASSWORD` before the first production start.

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

## CSV Bulk Upload

CSV bulk upload is available to admin users from:

```text
Settings → Import Employees / Import Users / Import Roster
```

Each import card lets you download a CSV template, upload a filled CSV file, preview the parsed rows, and submit the import.

Employee CSV:

```csv
name,emp_code,job_title,email,phone
Alice Johnson,STPL1001,Support Engineer,alice@example.com,9876543210
```

User CSV:

```csv
name,username,password,role,team_name
Alice Johnson,alice,changeme123,member,Support Alpha
```

Roster CSV uses the original grid format:

```csv
name,emp_code,team_name,month,1,2,3,...,31
Aditya,STPL1206,Support Alpha,2026-05,GS,AS,AS,...
```

Rules:

- User `role` must be `admin` or `member`.
- User passwords must be at least 8 characters.
- `team_name` must match an existing team exactly.
- Roster `emp_code` must match an existing employee.
- Roster shift codes must be one of `MS`, `GS`, `AS`, `NS`, `WO`, `EL`.
- Blank roster day cells are skipped.

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
INITIAL_ADMIN_PASSWORD=<first-admin-password> \
PORT=3001 \
./bin/roster-api
```

Run the frontend:

```bash
cd frontend
GO_API_BASE=http://localhost:3001 npm start
```

In production, run both processes behind a reverse proxy or process manager. Route user traffic to the Next.js app and make sure `/api/*` reaches the Go backend.

## Auto-Start After Reboot With Systemd

Systemd is the recommended option for a Linux server because it starts services on boot, restarts crashed processes, captures logs with `journalctl`, and does not require a separate Node process manager.

Service templates are provided in:

```text
deploy/systemd/
├── roster-api.service.example
├── roster-api.env.example
├── roster-web.service.example
└── roster-web.env.example
```

The templates assume this production layout:

| Path | Purpose |
|---|---|
| `/opt/roster-dashboard` | Deployed project directory |
| `/opt/roster-dashboard/bin/roster-api` | Built Go backend binary |
| `/opt/roster-dashboard/frontend` | Built Next.js frontend |
| `/etc/roster-dashboard` | Environment files |
| `/var/lib/roster-dashboard/roster.db` | SQLite database |
| `roster` | Linux user/group running the services |

Edit the `.service` and `.env` files if your server uses different paths, ports, or user names.

### 1. Build The App

From the project root:

```bash
go build -o bin/roster-api ./cmd/server

cd frontend
npm install
npm run build
cd ..
```

### 2. Copy Files To The Server Layout

Example:

```bash
sudo useradd --system --home /opt/roster-dashboard --shell /usr/sbin/nologin roster
sudo mkdir -p /opt/roster-dashboard /etc/roster-dashboard /var/lib/roster-dashboard
sudo cp -R . /opt/roster-dashboard
sudo chown -R roster:roster /opt/roster-dashboard /var/lib/roster-dashboard
```

If you already have a database, copy it to the path you choose for `DB_PATH`.

### 3. Install Environment Files

```bash
sudo cp deploy/systemd/roster-api.env.example /etc/roster-dashboard/roster-api.env
sudo cp deploy/systemd/roster-web.env.example /etc/roster-dashboard/roster-web.env
```

Then edit both files:

```bash
sudo nano /etc/roster-dashboard/roster-api.env
sudo nano /etc/roster-dashboard/roster-web.env
```

At minimum, replace `SESSION_SECRET` and `INITIAL_ADMIN_PASSWORD` before the first production start. Production rejects signing secrets shorter than 32 characters. Generate a strong session secret with:

```bash
openssl rand -hex 32
```

### 4. Install Systemd Units

```bash
sudo cp deploy/systemd/roster-api.service.example /etc/systemd/system/roster-api.service
sudo cp deploy/systemd/roster-web.service.example /etc/systemd/system/roster-web.service
sudo systemctl daemon-reload
```

### 5. Enable And Start

```bash
sudo systemctl enable --now roster-api
sudo systemctl enable --now roster-web
```

Check status:

```bash
sudo systemctl status roster-api
sudo systemctl status roster-web
```

View logs:

```bash
journalctl -u roster-api -f
journalctl -u roster-web -f
```

Restart after changes:

```bash
sudo systemctl restart roster-api
sudo systemctl restart roster-web
```

### Other Viable Options

- **Docker Compose**: good if you want repeatable deployments and isolated runtime versions.
- **PM2**: works well for the Next.js frontend, but it does not manage the Go backend as naturally as systemd.
- **Supervisor**: simple process supervision, but systemd is usually already available on modern Linux servers.

## Environment Variables

| Variable | Default | Used by | Description |
|---|---|---|---|
| `PORT` | `3001` | Backend | HTTP listen port |
| `DB_PATH` | `./roster.db` | Backend | SQLite database path |
| `SESSION_SECRET` | development fallback | Backend | HS256 token/session signing secret; at least 32 characters in production |
| `JWT_SECRET` | optional | Backend | Used if `SESSION_SECRET` is unset |
| `APP_ENV` | `development` | Backend | Set to `production` to require a real secret |
| `INITIAL_ADMIN_USERNAME` | `admin` | Backend | Username used only when seeding the first admin |
| `INITIAL_ADMIN_NAME` | `Admin` | Backend | Display name used only when seeding the first admin |
| `INITIAL_ADMIN_PASSWORD` | `admin123` in development, required in production | Backend | Password used only when seeding the first admin |
| `GO_API_BASE` | `http://localhost:3001` | Frontend | Backend URL used by Next.js rewrites |

## Notes

- API routes are exposed under `/api`.
- `/api/auth/login` sets an HttpOnly session cookie and returns the user payload. It also returns a Bearer token for API compatibility, but the frontend does not store it.
- The Next.js UI includes the dashboard, roster grid, admin panel, settings, dark mode, and CSV import workflows.
- SQLite is a single-writer database; keep backend deployment to one process unless you intentionally design around that limit.
