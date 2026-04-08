# Replicated SDK Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate the Replicated SDK sidecar to enforce license entitlements, show expiry/update banners, and report custom metrics.

**Architecture:** The SDK runs as a sidecar container at `http://localhost:3000` (configurable via `SDK_SERVICE_URL`). GameShelf calls it at startup and on each request via a thin client with in-process caching. All SDK calls are fail-open: if the SDK is unavailable (local dev, test), GameShelf continues normally. Metrics are emitted in a background goroutine every 60 seconds.

**Tech Stack:** Go stdlib `net/http`, `sync.RWMutex` for caching, existing Chi middleware pattern, Go templates for banners.

---

## File Map

| File | Change |
|------|--------|
| `internal/sdk/client.go` | New — HTTP client, timeouts, fail-open helper |
| `internal/sdk/license.go` | New — `GetLicenseInfo`, `GetFieldValue`, `IsFeatureEnabled`, `IsLicenseValid`, 1-min cache |
| `internal/sdk/updates.go` | New — `CheckForUpdates`, 5-min cache |
| `internal/sdk/metrics.go` | New — `ReportMetrics`, `RunMetricsLoop` (background goroutine) |
| `internal/config/config.go` | Modify — add `SDKServiceURL` field |
| `internal/api/server.go` | Modify — add `sdk *sdk.Client` field, wire into `NewServer` |
| `internal/api/handlers.go` | Modify — add banner fields to `PageData`, populate in `pageBase` |
| `internal/api/middleware.go` | New — `sdkAdminGateMiddleware` |
| `templates/base.html` | Modify — add license expiry and update available banners |
| `cmd/gameshelf/main.go` | Modify — init SDK client, start metrics goroutine |

---

### Task 1: SDK package — HTTP client

**Files:**
- Create: `internal/sdk/client.go`

- [ ] **Step 1: Write the file**

```go
package sdk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is a thin wrapper around the Replicated SDK sidecar HTTP API.
// All methods are fail-open: if the SDK is unreachable they return zero
// values and a nil error so the application continues normally.
type Client struct {
	base       string
	httpClient *http.Client
}

// New returns a Client pointed at baseURL (e.g. "http://localhost:3000").
// Pass an empty string to get a no-op client that always returns zero values.
func New(baseURL string) *Client {
	return &Client{
		base: baseURL,
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Available reports whether the SDK sidecar is configured.
func (c *Client) Available() bool {
	return c.base != ""
}

// get performs a GET request and JSON-decodes the response body into dst.
// Returns an error only for non-2xx responses; connection failures return nil
// so callers stay fail-open (they check Available() first).
func (c *Client) get(ctx context.Context, path string, dst any) error {
	if !c.Available() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path, nil)
	if err != nil {
		return fmt.Errorf("sdk: build request: %w", err)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		// Connection failure — stay fail-open, return nil
		return nil
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("sdk: %s returned %d: %s", path, resp.StatusCode, body)
	}
	if dst != nil {
		if err := json.Unmarshal(body, dst); err != nil {
			return fmt.Errorf("sdk: decode %s: %w", path, err)
		}
	}
	return nil
}

// patch performs a PATCH request with a JSON body.
func (c *Client) patch(ctx context.Context, path string, body io.Reader) error {
	if !c.Available() {
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPatch, c.base+path, body)
	if err != nil {
		return fmt.Errorf("sdk: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil // fail-open
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sdk: PATCH %s returned %d: %s", path, resp.StatusCode, b)
	}
	return nil
}
```

- [ ] **Step 2: Verify it compiles**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./internal/sdk/...
```
Expected: no output.

- [ ] **Step 3: Commit**

```bash
git add internal/sdk/client.go
git commit -m "feat(sdk): add HTTP client with fail-open semantics"
```

---

### Task 2: SDK package — license info & caching

**Files:**
- Create: `internal/sdk/license.go`

- [ ] **Step 1: Write the file**

```go
package sdk

import (
	"context"
	"sync"
	"time"
)

// LicenseInfo mirrors the /api/v1/license/info response.
type LicenseInfo struct {
	LicenseID      string     `json:"licenseID"`
	LicenseType    string     `json:"licenseType"`
	CustomerName   string     `json:"customerName"`
	ExpirationDate *time.Time `json:"expirationDate"`
	IsExpired      bool       `json:"isExpired"`
}

// FieldValue mirrors the /api/v1/license/fields/:name response.
type FieldValue struct {
	Value string `json:"value"`
}

// licenseCache holds a cached LicenseInfo with a TTL.
type licenseCache struct {
	mu        sync.RWMutex
	info      *LicenseInfo
	fetchedAt time.Time
	ttl       time.Duration
}

var globalLicenseCache = &licenseCache{ttl: 1 * time.Minute}

// GetLicenseInfo returns license info, using a 1-minute cache.
// Returns nil, nil when SDK is unavailable (fail-open).
func (c *Client) GetLicenseInfo(ctx context.Context) (*LicenseInfo, error) {
	if !c.Available() {
		return nil, nil
	}

	globalLicenseCache.mu.RLock()
	if globalLicenseCache.info != nil && time.Since(globalLicenseCache.fetchedAt) < globalLicenseCache.ttl {
		info := globalLicenseCache.info
		globalLicenseCache.mu.RUnlock()
		return info, nil
	}
	globalLicenseCache.mu.RUnlock()

	var info LicenseInfo
	if err := c.get(ctx, "/api/v1/license/info", &info); err != nil {
		return nil, err
	}

	globalLicenseCache.mu.Lock()
	globalLicenseCache.info = &info
	globalLicenseCache.fetchedAt = time.Now()
	globalLicenseCache.mu.Unlock()

	return &info, nil
}

// GetFieldValue returns the value of a named license field.
// Returns "", nil when SDK is unavailable.
func (c *Client) GetFieldValue(ctx context.Context, fieldName string) (string, error) {
	if !c.Available() {
		return "", nil
	}
	var fv FieldValue
	if err := c.get(ctx, "/api/v1/license/fields/"+fieldName, &fv); err != nil {
		return "", err
	}
	return fv.Value, nil
}

// IsFeatureEnabled returns true if the named license field equals "true".
// Returns true when SDK is unavailable (fail-open).
func (c *Client) IsFeatureEnabled(ctx context.Context, fieldName string) bool {
	val, err := c.GetFieldValue(ctx, fieldName)
	if err != nil || val == "" {
		return true // fail-open
	}
	return val == "true"
}

// IsLicenseValid returns true if the license exists and is not expired.
// Returns true when SDK is unavailable (fail-open).
func (c *Client) IsLicenseValid(ctx context.Context) bool {
	info, err := c.GetLicenseInfo(ctx)
	if err != nil || info == nil {
		return true // fail-open
	}
	return !info.IsExpired
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./internal/sdk/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/sdk/license.go
git commit -m "feat(sdk): license info fetching with 1-minute cache"
```

---

### Task 3: SDK package — update checks

**Files:**
- Create: `internal/sdk/updates.go`

- [ ] **Step 1: Write the file**

```go
package sdk

import (
	"context"
	"sync"
	"time"
)

// UpdateInfo describes a pending application update.
type UpdateInfo struct {
	VersionLabel string `json:"versionLabel"`
	CreatedAt    string `json:"createdAt"`
}

// UpdatesResponse mirrors the /api/v1/app/updates response.
type UpdatesResponse struct {
	Updates []UpdateInfo `json:"updates"`
}

type updatesCache struct {
	mu        sync.RWMutex
	updates   []UpdateInfo
	fetchedAt time.Time
	ttl       time.Duration
}

var globalUpdatesCache = &updatesCache{ttl: 5 * time.Minute}

// CheckForUpdates returns available updates, using a 5-minute cache.
// Returns nil, nil when SDK is unavailable.
func (c *Client) CheckForUpdates(ctx context.Context) ([]UpdateInfo, error) {
	if !c.Available() {
		return nil, nil
	}

	globalUpdatesCache.mu.RLock()
	if time.Since(globalUpdatesCache.fetchedAt) < globalUpdatesCache.ttl {
		updates := globalUpdatesCache.updates
		globalUpdatesCache.mu.RUnlock()
		return updates, nil
	}
	globalUpdatesCache.mu.RUnlock()

	var resp UpdatesResponse
	if err := c.get(ctx, "/api/v1/app/updates", &resp); err != nil {
		return nil, err
	}

	globalUpdatesCache.mu.Lock()
	globalUpdatesCache.updates = resp.Updates
	globalUpdatesCache.fetchedAt = time.Now()
	globalUpdatesCache.mu.Unlock()

	return resp.Updates, nil
}

// HasUpdate returns true if there is at least one pending update.
// Returns false when SDK is unavailable.
func (c *Client) HasUpdate(ctx context.Context) bool {
	updates, err := c.CheckForUpdates(ctx)
	if err != nil || len(updates) == 0 {
		return false
	}
	return true
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./internal/sdk/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/sdk/updates.go
git commit -m "feat(sdk): update availability check with 5-minute cache"
```

---

### Task 4: SDK package — custom metrics

**Files:**
- Create: `internal/sdk/metrics.go`

- [ ] **Step 1: Write the file**

```go
package sdk

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"time"
)

// Metric is a single key/value pair for the custom metrics API.
type Metric struct {
	Key   string `json:"key"`
	Value any    `json:"value"`
}

// metricsPayload is the request body for PATCH /api/v1/app/custom-metrics.
type metricsPayload struct {
	Data []Metric `json:"data"`
}

// ReportMetrics sends the provided metrics to the SDK sidecar.
func (c *Client) ReportMetrics(ctx context.Context, metrics []Metric) error {
	payload := metricsPayload{Data: metrics}
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return c.patch(ctx, "/api/v1/app/custom-metrics", bytes.NewReader(b))
}

// RunMetricsLoop starts a background goroutine that queries the database every
// interval and reports custom metrics to the SDK. It exits when ctx is cancelled.
func (c *Client) RunMetricsLoop(ctx context.Context, db *sql.DB, interval time.Duration) {
	if !c.Available() {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.collectAndReport(ctx, db); err != nil {
					log.Printf("sdk: metrics report error: %v", err)
				}
			}
		}
	}()
}

// collectAndReport queries DB counters and sends them to the SDK.
func (c *Client) collectAndReport(ctx context.Context, db *sql.DB) error {
	var scoresSubmitted int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM scores`).Scan(&scoresSubmitted); err != nil {
		return err
	}
	var gamesPlayed int64
	if err := db.QueryRowContext(ctx, `SELECT COUNT(DISTINCT game_slug) FROM scores`).Scan(&gamesPlayed); err != nil {
		return err
	}
	var activePlayers int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT player_id) FROM scores
		WHERE played_at > NOW() - INTERVAL '24 hours'`).Scan(&activePlayers); err != nil {
		return err
	}
	var activeGames int64
	if err := db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT game_slug) FROM scores
		WHERE played_at > NOW() - INTERVAL '24 hours'`).Scan(&activeGames); err != nil {
		return err
	}

	metrics := []Metric{
		{Key: "games_played_total", Value: gamesPlayed},
		{Key: "scores_submitted_total", Value: scoresSubmitted},
		{Key: "active_players_24h", Value: activePlayers},
		{Key: "active_games", Value: activeGames},
	}
	return c.ReportMetrics(ctx, metrics)
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./internal/sdk/...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/sdk/metrics.go
git commit -m "feat(sdk): custom metrics background reporter"
```

---

### Task 5: Config — add SDK_SERVICE_URL

**Files:**
- Modify: `internal/config/config.go`

- [ ] **Step 1: Update config.go**

```go
package config

import "os"

type Config struct {
	DatabaseURL    string
	RedisURL       string
	AdminSecret    string
	Port           string
	SiteName       string
	IdentitySecret string // optional; auto-generated and stored in DB if empty
	SDKServiceURL  string // URL of Replicated SDK sidecar, e.g. http://localhost:3000
}

func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "GameShelf"
	}
	adminSecret := os.Getenv("ADMIN_SECRET")
	if adminSecret == "" {
		adminSecret = "changeme"
	}
	return Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		AdminSecret:    adminSecret,
		Port:           port,
		SiteName:       siteName,
		IdentitySecret: os.Getenv("IDENTITY_SECRET"),
		SDKServiceURL:  os.Getenv("SDK_SERVICE_URL"),
	}
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/config/config.go
git commit -m "feat(config): add SDK_SERVICE_URL config field"
```

---

### Task 6: Wire SDK into Server and main

**Files:**
- Modify: `internal/api/server.go`
- Modify: `cmd/gameshelf/main.go`

- [ ] **Step 1: Update server.go**

```go
package api

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/gameshelf/gameshelf/internal/config"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
	"github.com/gameshelf/gameshelf/internal/sdk"
)

// Server holds all application dependencies.
type Server struct {
	db       *sql.DB
	lb       *leaderboard.Client
	sdk      *sdk.Client
	tmpls    map[string]*template.Template
	staticFS fs.FS
	cfg      config.Config
}

// pageNames lists the templates that can be rendered.
var pageNames = []string{"index.html", "game.html", "leaderboard.html", "admin.html"}

// NewServer constructs a Server, parsing each page template together with base.html.
func NewServer(db *sql.DB, lb *leaderboard.Client, sdkClient *sdk.Client, templatesFS embed.FS, staticFS embed.FS, cfg config.Config) (*Server, error) {
	tmpls := make(map[string]*template.Template, len(pageNames))
	for _, page := range pageNames {
		t, err := template.ParseFS(templatesFS, "templates/base.html", "templates/"+page)
		if err != nil {
			return nil, fmt.Errorf("parsing template %s: %w", page, err)
		}
		tmpls[page] = t
	}
	stripped, err := fs.Sub(staticFS, "static")
	if err != nil {
		return nil, fmt.Errorf("sub static fs: %w", err)
	}
	return &Server{db: db, lb: lb, sdk: sdkClient, tmpls: tmpls, staticFS: stripped, cfg: cfg}, nil
}

// render executes a named template with buffering to prevent partial responses.
func (s *Server) render(w http.ResponseWriter, name string, data PageData) {
	t, ok := s.tmpls[name]
	if !ok {
		http.Error(w, "unknown template: "+name, http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, name, data); err != nil {
		http.Error(w, "template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	buf.WriteTo(w) //nolint:errcheck
}

// Handler builds and returns the root http.Handler.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Static files served at /static/*
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.FS(s.staticFS))))

	// Public pages
	r.Get("/", s.indexHandler)
	r.Get("/games/{slug}", s.gameHandler)
	r.Get("/leaderboard/{slug}", s.leaderboardHandler)
	r.Get("/logo", s.logoHandler)

	// Score API
	r.Post("/api/scores", s.submitScoreHandler)
	r.Get("/api/scores/{slug}", s.getScoresHandler)

	// Admin (protected by auth + SDK entitlement gate)
	r.Group(func(r chi.Router) {
		r.Use(s.adminAuthMiddleware)
		r.Use(s.sdkAdminGateMiddleware)
		r.Get("/admin", s.adminHandler)
		r.Post("/admin/games/{slug}/toggle", s.toggleGameHandler)
		r.Post("/admin/branding", s.updateBrandingHandler)
		r.Post("/admin/logo", s.uploadLogoHandler)
		r.Post("/admin/identity/regenerate", s.regenerateIdentitySecretHandler)
	})

	// Health
	r.Get("/healthz", s.healthHandler)

	return r
}
```

- [ ] **Step 2: Update main.go**

```go
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"time"

	gameshelf "github.com/gameshelf/gameshelf"
	"github.com/gameshelf/gameshelf/internal/api"
	"github.com/gameshelf/gameshelf/internal/config"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
	"github.com/gameshelf/gameshelf/internal/sdk"
)

func main() {
	cfg := config.Load()

	if cfg.DatabaseURL == "" {
		log.Fatal("DATABASE_URL environment variable is required")
	}
	if cfg.RedisURL == "" {
		log.Fatal("REDIS_URL environment variable is required")
	}

	// Connect to PostgreSQL
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("connecting to database: %v", err)
	}
	defer database.Close()

	// Wait for DB with retry
	log.Println("Waiting for database to be ready...")
	if err := waitForDB(database, 30, 2*time.Second); err != nil {
		log.Fatalf("database never became available: %v", err)
	}
	log.Println("Database is ready.")

	// Run migrations
	if err := db.Migrate(database, gameshelf.MigrationsFS); err != nil {
		log.Fatalf("running migrations: %v", err)
	}
	log.Println("Migrations applied.")

	// Connect to Redis
	lb, err := leaderboard.New(cfg.RedisURL)
	if err != nil {
		log.Fatalf("connecting to redis: %v", err)
	}
	log.Println("Waiting for Redis to be ready...")
	if err := waitForRedis(lb, 30, 2*time.Second); err != nil {
		log.Fatalf("redis never became available: %v", err)
	}
	log.Println("Redis is ready.")

	// Seed Redis leaderboards from PostgreSQL
	if err := seedLeaderboards(database, lb); err != nil {
		log.Printf("Warning: could not seed leaderboards: %v", err)
	}

	// Initialize Replicated SDK client (fail-open if SDK_SERVICE_URL is unset)
	sdkClient := sdk.New(cfg.SDKServiceURL)
	if sdkClient.Available() {
		log.Printf("Replicated SDK connected at %s", cfg.SDKServiceURL)
	} else {
		log.Println("Replicated SDK not configured (SDK_SERVICE_URL unset) — license checks disabled")
	}

	// Build HTTP server
	srv, err := api.NewServer(database, lb, sdkClient, gameshelf.TemplatesFS, gameshelf.StaticFS, cfg)
	if err != nil {
		log.Fatalf("creating server: %v", err)
	}

	// Start custom metrics background reporter
	ctx := context.Background()
	sdkClient.RunMetricsLoop(ctx, database, 60*time.Second)

	addr := ":" + cfg.Port
	log.Printf("listening on %s", addr)
	httpSrv := &http.Server{
		Addr:         addr,
		Handler:      srv.Handler(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
	if err := httpSrv.ListenAndServe(); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// waitForRedis retries pinging Redis until available.
func waitForRedis(lb *leaderboard.Client, maxRetries int, interval time.Duration) error {
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := lb.Ping(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("Redis not ready (attempt %d/%d): %v", i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("redis not ready after %d attempts", maxRetries)
}

// waitForDB retries pinging the database until available.
func waitForDB(d *sql.DB, maxRetries int, interval time.Duration) error {
	for i := 1; i <= maxRetries; i++ {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		err := d.PingContext(ctx)
		cancel()
		if err == nil {
			return nil
		}
		log.Printf("DB not ready (attempt %d/%d): %v", i, maxRetries, err)
		if i < maxRetries {
			time.Sleep(interval)
		}
	}
	return fmt.Errorf("database not ready after %d attempts", maxRetries)
}

// seedLeaderboards loads PostgreSQL scores into Redis sorted sets on startup.
func seedLeaderboards(d *sql.DB, lb *leaderboard.Client) error {
	rows, err := d.Query(`
		SELECT s.game_slug, p.display_name, s.score, s.played_at
		FROM scores s
		JOIN players p ON p.id = s.player_id
		ORDER BY s.game_slug, s.played_at`)
	if err != nil {
		return err
	}
	defer rows.Close()

	byGame := make(map[string][]leaderboard.SeedRow)
	for rows.Next() {
		var r leaderboard.SeedRow
		if err := rows.Scan(&r.GameSlug, &r.PlayerName, &r.Score, &r.PlayedAt); err != nil {
			return err
		}
		byGame[r.GameSlug] = append(byGame[r.GameSlug], r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	for gameSlug, seedRows := range byGame {
		if err := lb.SeedGame(ctx, gameSlug, seedRows); err != nil {
			log.Printf("Warning: seeding %s leaderboard: %v", gameSlug, err)
		}
	}
	log.Printf("Leaderboard seeding complete (%d games).", len(byGame))
	return nil
}
```

- [ ] **Step 3: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/api/server.go cmd/gameshelf/main.go
git commit -m "feat(sdk): wire SDK client into server and start metrics loop"
```

---

### Task 7: Admin entitlement gate middleware

**Files:**
- Create: `internal/api/middleware.go`

- [ ] **Step 1: Find where adminAuthMiddleware lives**

```bash
grep -rn "adminAuthMiddleware" /Users/adam/gt/GameShelf/mayor/rig/internal/api/
```

- [ ] **Step 2: Write the file**

```go
package api

import (
	"log"
	"net/http"
)

// sdkAdminGateMiddleware blocks access to admin routes when the
// admin_panel_enabled license field is explicitly set to "false".
// Fail-open: if SDK is unavailable or field is absent, access is allowed.
func (s *Server) sdkAdminGateMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !s.sdk.IsFeatureEnabled(r.Context(), "admin_panel_enabled") {
			log.Printf("admin: access denied by license entitlement (admin_panel_enabled=false)")
			http.Error(w, "Admin panel disabled by license", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
```

- [ ] **Step 3: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./...
```
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add internal/api/middleware.go
git commit -m "feat(sdk): admin panel entitlement gate via license field"
```

---

### Task 8: Banner fields in PageData and pageBase

**Files:**
- Modify: `internal/api/handlers.go`

- [ ] **Step 1: Update handlers.go**

```go
package api

import (
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/gameshelf/gameshelf/internal/db"
	"github.com/gameshelf/gameshelf/internal/leaderboard"
)

// PageData is passed to every template render.
type PageData struct {
	SiteName        string
	PrimaryColor    string
	SecondaryColor  string
	BackgroundColor string
	FontFamily      string
	HasLogo         bool
	PageTitle       string
	Token           string // admin token, preserved across form POSTs
	PlayerName      string // pre-filled player name from cookie / identity token
	// SDK-driven banners
	LicenseExpired      bool // true → red "license expired" banner
	LicenseExpiringSoon bool // true → yellow "expiring soon" banner (< 30 days)
	UpdateAvailable     bool // true → blue "update available" banner
	// Page-specific (only one populated per page)
	Games                []db.Game
	Game                 *db.Game
	Scores               []leaderboard.Entry
	DBScores             []db.Score
	AllGames             []db.Game
	Site                 *db.Site
	IdentitySecretMasked string // shown (masked) on admin panel
}

// pageBase fills the branding fields from the DB and SDK banner state.
func (s *Server) pageBase(r *http.Request) PageData {
	site, err := db.GetSite(s.db)
	var data PageData
	if err != nil || site == nil {
		data = PageData{
			SiteName:        s.cfg.SiteName,
			PrimaryColor:    "#3B82F6",
			SecondaryColor:  "#1E40AF",
			BackgroundColor: "#F9FAFB",
			FontFamily:      "system",
		}
	} else {
		data = PageData{
			SiteName:        site.Name,
			PrimaryColor:    site.PrimaryColor,
			SecondaryColor:  site.SecondaryColor,
			BackgroundColor: site.BackgroundColor,
			FontFamily:      site.FontFamily,
			HasLogo:         site.HasLogo,
			Site:            site,
		}
	}

	// Populate SDK banners (fail-open: errors are logged and ignored)
	if s.sdk.Available() {
		if info, err := s.sdk.GetLicenseInfo(r.Context()); err == nil && info != nil {
			if info.IsExpired {
				data.LicenseExpired = true
			} else if info.ExpirationDate != nil && time.Until(*info.ExpirationDate) < 30*24*time.Hour {
				data.LicenseExpiringSoon = true
			}
		}
		data.UpdateAvailable = s.sdk.HasUpdate(r.Context())
	}

	return data
}

// GET / — game library landing page
func (s *Server) indexHandler(w http.ResponseWriter, r *http.Request) {
	games, err := db.GetEnabledGames(s.db)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	data := s.pageBase(r)
	data.PageTitle = "Game Library"
	data.Games = games
	s.render(w, "index.html", data)
}

// GET /games/:slug — play a specific game
func (s *Server) gameHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	game, err := db.GetGame(s.db, slug)
	if err != nil || game == nil || !game.Enabled {
		http.NotFound(w, r)
		return
	}

	// Resolve player name: identity token takes precedence over existing cookie.
	playerName := getPlayerFromCookie(r)
	if token := r.URL.Query().Get("gs_identity"); token != "" {
		if secret, err := s.getOrCreateIdentitySecret(); err == nil {
			if name, ok := verifyIdentityToken(secret, token); ok {
				setPlayerCookie(w, name)
				playerName = name
			}
		}
	}

	data := s.pageBase(r)
	data.PageTitle = game.Name
	data.Game = game
	data.PlayerName = playerName
	s.render(w, "game.html", data)
}

// GET /leaderboard/:slug — leaderboard for a game
func (s *Server) leaderboardHandler(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	game, err := db.GetGame(s.db, slug)
	if err != nil || game == nil {
		http.NotFound(w, r)
		return
	}
	entries, err := s.lb.TopScores(r.Context(), slug, 50)
	if err != nil {
		log.Printf("leaderboard redis fallback for %s: %v", slug, err)
		// Fallback to DB if Redis unavailable
		dbScores, dbErr := db.GetTopScores(s.db, slug, 50)
		if dbErr != nil {
			log.Printf("leaderboard db fallback also failed for %s: %v", slug, dbErr)
		}
		entries = make([]leaderboard.Entry, len(dbScores))
		for i, sc := range dbScores {
			entries[i] = leaderboard.Entry{Rank: i + 1, PlayerName: sc.PlayerName, Score: sc.Score}
		}
	}
	data := s.pageBase(r)
	data.PageTitle = game.Name + " Leaderboard"
	data.Game = game
	data.Scores = entries
	s.render(w, "leaderboard.html", data)
}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add internal/api/handlers.go
git commit -m "feat(sdk): add banner fields to PageData, populate from SDK in pageBase"
```

---

### Task 9: Banners in base.html template

**Files:**
- Modify: `templates/base.html`

- [ ] **Step 1: Update base.html**

Replace the entire file with:

```html
{{ define "base.html" }}
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8" />
  <meta name="viewport" content="width=device-width, initial-scale=1.0" />
  <title>{{ if .PageTitle }}{{ .PageTitle }} — {{ end }}{{ .SiteName }}</title>
  <script src="https://cdn.tailwindcss.com"></script>
  <style>
    :root {
      --color-primary:    {{ .PrimaryColor }};
      --color-secondary:  {{ .SecondaryColor }};
      --color-background: {{ .BackgroundColor }};
    }
    {{ if eq .FontFamily "serif" }}
    body { font-family: Georgia, 'Times New Roman', serif; }
    {{ else if eq .FontFamily "mono" }}
    body { font-family: 'Courier New', Courier, monospace; }
    {{ else }}
    body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; }
    {{ end }}
  </style>
</head>
<body style="background-color: {{ .BackgroundColor }}" class="min-h-screen flex flex-col">
  <header style="background-color: {{ .PrimaryColor }}">
    <nav class="max-w-6xl mx-auto px-4 py-4 flex items-center justify-between">
      <a href="/" class="text-white text-2xl font-bold tracking-tight flex items-center gap-3">
        {{ if .HasLogo }}<img src="/logo" alt="{{ .SiteName }}" class="h-8 w-auto">{{ end }}
        <span>{{ .SiteName }}</span>
      </a>
      <div class="flex items-center gap-4">
        <a href="/" class="text-white/80 hover:text-white text-sm transition-colors">Games</a>
        <a href="/admin" class="text-white/60 hover:text-white/90 text-xs transition-colors">Admin</a>
      </div>
    </nav>
  </header>

  {{- if .LicenseExpired }}
  <div class="bg-red-600 text-white text-center py-2 px-4 text-sm font-medium">
    Your GameShelf license has expired. Please renew to continue using this software.
  </div>
  {{- else if .LicenseExpiringSoon }}
  <div class="bg-yellow-400 text-yellow-900 text-center py-2 px-4 text-sm font-medium">
    Your GameShelf license expires soon. Please renew to avoid interruption.
  </div>
  {{- end }}
  {{- if .UpdateAvailable }}
  <div class="bg-blue-600 text-white text-center py-2 px-4 text-sm font-medium">
    A new version of GameShelf is available. Contact your administrator to update.
  </div>
  {{- end }}

  <main class="flex-1 max-w-6xl mx-auto px-4 py-8 w-full">
    {{ template "content" . }}
  </main>

  <footer class="border-t bg-white mt-16 py-6 text-center text-gray-400 text-sm">
    Powered by <strong>GameShelf</strong>
  </footer>
</body>
</html>
{{ end }}
```

- [ ] **Step 2: Compile check**

```bash
cd /Users/adam/gt/GameShelf/mayor/rig && go build ./...
```
Expected: no errors.

- [ ] **Step 3: Commit**

```bash
git add templates/base.html
git commit -m "feat(sdk): add license expiry and update available banners to base template"
```

---

## Self-Review

### Spec coverage

| Requirement | Task |
|-------------|------|
| `internal/sdk` package, HTTP client, fail-open | Task 1 |
| License info with 1-minute cache | Task 2 |
| `IsFeatureEnabled`, `IsLicenseValid` | Task 2 |
| Update check with 5-minute cache | Task 3 |
| Custom metrics goroutine, 60s interval | Task 4 |
| `games_played_total`, `scores_submitted_total`, `active_players_24h`, `active_games` | Task 4 |
| `SDK_SERVICE_URL` config | Task 5 |
| Fail-open when SDK unavailable | Tasks 1–4, 7, 8 |
| `admin_panel_enabled` entitlement gate on admin routes | Tasks 6, 7 |
| Expired → red banner | Tasks 8, 9 |
| Expiring < 30 days → yellow banner | Tasks 8, 9 |
| Update available → blue banner | Tasks 8, 9 |
| Wire SDK client into server | Task 6 |
| Start metrics goroutine in main | Task 6 |

### Type consistency

- `sdk.Client` type used identically in all tasks
- `PageData.LicenseExpired`, `LicenseExpiringSoon`, `UpdateAvailable` match template `{{- if .LicenseExpired }}` etc.
- `api.NewServer` third arg is `*sdk.Client` in Task 6, matches usage in Task 8
- `s.sdk.Available()`, `s.sdk.GetLicenseInfo()`, `s.sdk.HasUpdate()`, `s.sdk.IsFeatureEnabled()` all defined in Tasks 1–3

### No placeholders ✓
