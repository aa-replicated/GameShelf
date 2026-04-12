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

| Variable          | Default     | Description                                              |
|-------------------|-------------|----------------------------------------------------------|
| `DATABASE_URL`    | (required)  | PostgreSQL connection string                             |
| `REDIS_URL`       | (required)  | Redis connection string                                  |
| `ADMIN_SECRET`    | `changeme`  | Shared secret for /admin access                          |
| `PORT`            | `8080`      | HTTP listen port                                         |
| `SITE_NAME`       | `GameShelf` | Default site name                                        |
| `IDENTITY_SECRET` | (auto)      | HMAC secret for identity tokens; auto-generated if unset |

## API Reference

| Method | Path                          | Description                     |
|--------|-------------------------------|---------------------------------|
| GET    | `/`                           | Game library                    |
| GET    | `/games/:slug`                | Play a game                     |
| GET    | `/leaderboard/:slug`          | View leaderboard                |
| POST   | `/api/scores`                 | Submit a score                  |
| GET    | `/api/scores/:slug`           | Get leaderboard JSON            |
| GET    | `/admin`                           | Admin panel (auth required)     |
| POST   | `/admin/games/:slug/toggle`        | Enable/disable a game           |
| POST   | `/admin/branding`                  | Update site branding            |
| POST   | `/admin/logo`                      | Upload site logo                |
| POST   | `/admin/identity/regenerate`       | Regenerate identity secret      |
| GET    | `/healthz`                         | Health check (200 OK / 503)     |

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

### Player Identity

GameShelf uses **soft identity**: no accounts, no passwords. Players are recognised by a cookie set the first time they submit a score. If they return directly to `games.yoursite.com`, their name is pre-filled automatically.

#### Passing identity from your main site

If you want a logged-in user on your main site to arrive at GameShelf already named, generate a signed identity token and append it to any GameShelf link:

```
https://games.yoursite.com/games/snake?gs_identity=<token>
```

GameShelf verifies the token and sets the player-name cookie automatically. The token is HMAC-SHA256 signed and includes a 30-day expiry.

**Token format** (base64url-encoded):

```
playerName|expiryUnixSeconds|hmac-sha256-hex
```

**Generating a token** (Go example):

```go
import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/base64"
    "fmt"
    "strconv"
    "time"
)

func signIdentityToken(secret, playerName string) string {
    expiry := strconv.FormatInt(time.Now().Add(30*24*time.Hour).Unix(), 10)
    msg := playerName + "|" + expiry
    mac := hmac.New(sha256.New, []byte(secret))
    mac.Write([]byte(msg))
    raw := msg + "|" + fmt.Sprintf("%x", mac.Sum(nil))
    return base64.RawURLEncoding.EncodeToString([]byte(raw))
}
```

The `IDENTITY_SECRET` is displayed (masked) in the Admin panel under **Player Identity**. Copy the full secret from the database or set the `IDENTITY_SECRET` environment variable to use a fixed value. You can regenerate the secret from the Admin panel at any time — all existing tokens are immediately invalidated.

### HTTPS

GameShelf itself listens on plain HTTP. Terminate TLS at your reverse proxy, ingress controller, or load balancer and forward plain HTTP to GameShelf. If you're using the dedicated subdomain option, your proxy (nginx, Caddy, etc.) handles the certificate — Caddy does this automatically.

## Helm Install (Replicated)

Installing directly via Helm from the Replicated registry (not via KOTS/Embedded Cluster).

### Prerequisites

- A Replicated customer license ID (Vendor Portal → Customers → click customer → License ID)
- A Replicated customer email address
- `helm` v3.8+, `kubectl` pointed at your target cluster

### Install

```bash
# 1. Create the namespace
kubectl create namespace gameshelf

# 2. Create the image pull secret (required for proxied images)
kubectl create secret docker-registry enterprise-pull-secret \
  --docker-server=proxy.adamanthony.dev \
  --docker-username=<customer-email> \
  --docker-password=<license-id> \
  -n gameshelf

# 3. Log into the Replicated OCI registry
helm registry login registry.replicated.com \
  --username <customer-email> \
  --password <license-id>

# 4. Install
helm install gameshelf \
  oci://registry.replicated.com/gameshelf/unstable/gameshelf \
  --version <chart-version> \
  --namespace gameshelf \
  --set adminSecret=<your-admin-password> \
  --set "gameshelf-sdk.integration.licenseID=<license-id>" \
  --set "gameshelf-sdk.integration.enabled=true"
```

> The chart version for each release is visible in the Vendor Portal under Releases, or in the GitHub Actions run log.

### Upgrade

```bash
helm upgrade gameshelf \
  oci://registry.replicated.com/gameshelf/unstable/gameshelf \
  --version <new-chart-version> \
  --reuse-values
```

### Access the app

```bash
kubectl port-forward svc/gameshelf 8080:80 -n gameshelf
```

Then open http://localhost:8080. Admin panel: http://localhost:8080/admin?token=<your-admin-password>

### Common overrides

| Value | Default | Description |
|-------|---------|-------------|
| `adminSecret` | `changeme` | Admin panel password |
| `siteName` | `GameShelf` | Site name shown in the UI |
| `service.type` | `ClusterIP` | Set to `NodePort` or `LoadBalancer` to expose externally |
| `service.nodePort` | `""` | NodePort port number (e.g. `30080`) |
| `ingress.enabled` | `false` | Enable ingress |
| `ingress.host` | `""` | Hostname for ingress (required when enabled) |
| `postgresql.enabled` | `true` | Use embedded PostgreSQL; set to `false` for external DB |
| `redis.enabled` | `true` | Use embedded Redis; set to `false` for external Redis |
| `gameshelf-sdk.integration.licenseID` | `""` | License ID for SDK integration mode (direct Helm installs) |
| `gameshelf-sdk.integration.enabled` | `false` | Enable SDK integration mode (direct Helm installs) |

## Architecture

```
Browser  →  Go HTTP (Chi)  →  PostgreSQL (scores, games, branding)
                          →  Redis (leaderboard sorted sets)
```

Templates and static assets are embedded in the binary via `go:embed`.
The app waits for PostgreSQL on startup (30 retries × 2s) before serving traffic.
