# GameShelf

A self-hosted white-label browser games platform. Deploy it, point your readers at it,
and they get a branded game portal with leaderboards — think NYT Games but yours to own.

## Features

- **4 built-in games**: Word Search, Anagram, Snake, Bubble Shooter
- **Leaderboards**: Per-game top scores powered by Redis sorted sets, backed by PostgreSQL
- **Admin panel**: Manage game visibility, update site branding (name + colors)
- **White-label**: Configure site name and brand colors via the admin panel or env vars
- **Self-contained**: Single binary, all assets embedded — no external runtime dependencies

## Quick Start (Docker)

```bash
# Clone the repo
git clone <repo-url>
cd gameshelf

# Start everything (app + postgres + redis)
ADMIN_SECRET=mypassword make run
```

Open http://localhost:8080 to play games.
Open http://localhost:8080/admin?token=mypassword to manage the admin panel.

## Local Development

Prerequisites: Go 1.22+, a running PostgreSQL instance, a running Redis instance.

```bash
# Start dependencies with docker-compose (detached)
docker-compose up postgres redis -d

# Run the dev server
make dev
```

## Configuration

All configuration is via environment variables:

| Variable       | Default     | Description                          |
|----------------|-------------|--------------------------------------|
| `DATABASE_URL` | (required)  | PostgreSQL connection string         |
| `REDIS_URL`    | (required)  | Redis connection string              |
| `ADMIN_SECRET` | `changeme`  | Shared secret for /admin access      |
| `PORT`         | `8080`      | HTTP listen port                     |
| `SITE_NAME`    | `GameShelf` | Default site name                    |

## API Reference

| Method | Path                          | Description                     |
|--------|-------------------------------|---------------------------------|
| GET    | `/`                           | Game library                    |
| GET    | `/games/:slug`                | Play a game                     |
| GET    | `/leaderboard/:slug`          | View leaderboard                |
| POST   | `/api/scores`                 | Submit a score                  |
| GET    | `/api/scores/:slug`           | Get leaderboard JSON            |
| GET    | `/admin`                      | Admin panel (auth required)     |
| POST   | `/admin/games/:slug/toggle`   | Enable/disable a game           |
| POST   | `/admin/branding`             | Update site branding            |
| GET    | `/healthz`                    | Health check (200 OK / 503)     |

### Score Submission

```http
POST /api/scores
Content-Type: application/json

{"game": "snake", "player_name": "Alice", "score": 340}
```

Response:
```json
{"ok": true, "rank": 3}
```

## Adding Games

1. Add a new row to the `games` table (or update `migrations/001_schema.sql`)
2. Create `static/games/<slug>.js`
3. The game must call `window.GameShelf.gameOver(score)` when the player finishes

## Architecture

```
Browser  →  Go HTTP (Chi)  →  PostgreSQL (scores, games, branding)
                          →  Redis (leaderboard sorted sets)
```

Templates and static assets are embedded in the binary via `go:embed`.
The app waits for PostgreSQL on startup (30 retries × 2s) before serving traffic.
