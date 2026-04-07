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

## Integrating Into an Existing Site

GameShelf is designed to run as a standalone service, but it fits naturally alongside an existing site. Here are the main integration patterns.

### Reverse Proxy with a Path Prefix

The most common setup: route a sub-path of your existing domain to GameShelf.

**nginx example** (games available at `https://yoursite.com/games/`):

```nginx
location /games/ {
    proxy_pass http://localhost:8080/;
    proxy_set_header Host $host;
    proxy_set_header X-Real-IP $remote_addr;
}
```

GameShelf doesn't need any path-prefix configuration — the reverse proxy strips the prefix before forwarding.

**Caddy example:**

```
yoursite.com {
    handle /games/* {
        uri strip_prefix /games
        reverse_proxy localhost:8080
    }
    # ... rest of your site
}
```

### Embedding via iframe

For lighter-weight integration — embed individual games or the full game library directly in your existing pages:

```html
<!-- Embed the full game library -->
<iframe src="https://games.yoursite.com/" width="100%" height="600" frameborder="0"></iframe>

<!-- Embed a specific game -->
<iframe src="https://games.yoursite.com/games/snake" width="500" height="520" frameborder="0"></iframe>
```

The game pages are self-contained and render cleanly inside an iframe.

### Custom Styles and Logo

Site branding is configured entirely through the Admin panel (`/admin`):

- **Primary color** — header background and accent color
- **Secondary color** — hover states and secondary accents
- **Background color** — page background
- **Font** — choose System (default), Serif, or Monospace

Changes take effect immediately for all visitors — no restart required.

To upload a logo, use the **Logo** section of the Admin panel. Supported formats: PNG, JPEG, GIF, WebP, SVG (max 2MB). Once uploaded, the logo appears in the navigation bar alongside the site name. The logo is stored in the database, so no file mounts or volume configuration is needed.

### Routing Considerations

- **All routes are served from the root** (`/`, `/games/:slug`, `/leaderboard/:slug`, `/admin`, `/api/*`, `/healthz`, `/static/*`). There are no sub-path prefixes built in — your reverse proxy should strip any prefix before forwarding.
- **Score submissions** go to `POST /api/scores`. If your existing site handles `/api` routes, make sure the proxy routes `/api/scores` and `/api/scores/*` to GameShelf.
- **Static assets** are embedded in the binary via `go:embed` and served from `/static/`. The exception is custom CSS or logo images mounted as volumes (see above), which are served from the same path.
- **Sessions**: GameShelf is stateless. The admin token is passed per-request (cookie, query param, or `Authorization` header). No session storage is required.
- **HTTPS**: GameShelf listens on plain HTTP. Terminate TLS at your reverse proxy or load balancer and forward plain HTTP to GameShelf.

## Architecture

```
Browser  →  Go HTTP (Chi)  →  PostgreSQL (scores, games, branding)
                          →  Redis (leaderboard sorted sets)
```

Templates and static assets are embedded in the binary via `go:embed`.
The app waits for PostgreSQL on startup (30 retries × 2s) before serving traffic.
