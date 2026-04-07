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

GameShelf runs as a **separate service with its own URL**. It is not embedded into your existing application's process or codebase — it runs alongside it, reachable at its own hostname. You then decide how to present it to your users.

### Step 1: Give GameShelf a URL

There are two ways to make GameShelf reachable:

#### Option A: Dedicated subdomain (recommended)

Point a subdomain at the machine running GameShelf. This is the simplest path and works for most deployments.

```
games.yoursite.com  →  GameShelf server (port 80/443)
```

Set up DNS: create an A record for `games.yoursite.com` pointing at the IP of the GameShelf machine. That's it — no changes to your existing site's server.

#### Option B: Route through your existing ingress or reverse proxy

If you already have a reverse proxy (nginx, Caddy, Traefik, etc.) or an ingress controller in front of your infrastructure, you can add a rule that forwards traffic to the GameShelf machine.

**nginx example** — route a subdomain to the GameShelf server:

```nginx
server {
    server_name games.yoursite.com;

    location / {
        proxy_pass http://<gameshelf-server-ip>:80;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

**Caddy example:**

```
games.yoursite.com {
    reverse_proxy <gameshelf-server-ip>:80
}
```

GameShelf doesn't need to know about your proxy — it just receives normal HTTP requests and responds to whatever hostname it receives.

> **Note:** GameShelf serves all routes from the root (`/`, `/games/:slug`, `/admin`, etc.). Sub-path routing (serving GameShelf at `yoursite.com/games/`) is not supported — use a dedicated hostname instead.

### Step 2: Present it to your users

Once GameShelf has a URL, you have two options for how users reach it:

#### Direct link / subdomain

Send users directly to `https://games.yoursite.com`. With white-label branding configured (logo, colors, font via the Admin panel), it looks like a natural part of your site even though it's a separate service.

#### Embed via iframe

Embed the full game library or individual games directly in your existing pages:

```html
<!-- Full game library -->
<iframe src="https://games.yoursite.com"
        width="100%" height="700" frameborder="0"
        allow="fullscreen"></iframe>

<!-- A specific game -->
<iframe src="https://games.yoursite.com/games/snake"
        width="500" height="540" frameborder="0"
        allow="fullscreen"></iframe>
```

The game pages are self-contained and render cleanly inside an iframe. Because GameShelf is on a separate origin (`games.yoursite.com`), the iframe has no access to your main site's cookies or storage — this is normal and expected browser security behavior.

### Branding

Configure GameShelf's appearance entirely through the Admin panel (`/admin`) — no code changes or restarts required:

- **Site name** — displayed in the nav bar
- **Logo** — upload PNG, JPEG, GIF, WebP, or SVG (max 2MB); stored in the database
- **Primary color** — header and accent color
- **Secondary color** — hover states
- **Background color** — page background
- **Font** — System (default), Serif, or Monospace

### HTTPS

GameShelf itself listens on plain HTTP. Terminate TLS at your reverse proxy, ingress controller, or load balancer and forward plain HTTP to GameShelf. If you're using the dedicated subdomain option, your proxy (nginx, Caddy, etc.) handles the certificate — Caddy does this automatically.

## Architecture

```
Browser  →  Go HTTP (Chi)  →  PostgreSQL (scores, games, branding)
                          →  Redis (leaderboard sorted sets)
```

Templates and static assets are embedded in the binary via `go:embed`.
The app waits for PostgreSQL on startup (30 retries × 2s) before serving traffic.
