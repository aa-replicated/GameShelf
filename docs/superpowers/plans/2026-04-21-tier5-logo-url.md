# Tier 5 Logo URL Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a `SITE_LOGO_URL` env var that lets an operator set a logo via URL through the KOTS config screen, satisfying Tier 5 rubric item 5.1 (second configurable app feature).

**Architecture:** Four layers of change. The Go backend gains a `SiteLogoURL` config field read from env and a `LogoURL` field on `PageData` populated in `pageBase()`. The HTML template renders the URL logo with fallback to the binary upload path. The Helm chart wires the new value through `values.yaml` → `configmap.yaml` → `deployment.yaml`. KOTS config and helmchart get the new field wired from the install screen.

**Tech Stack:** Go 1.21, Go `html/template`, Helm 3, KOTS Config (kots.io/v1beta1), KOTS HelmChart (kots.io/v1beta2)

---

## File Map

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `SiteLogoURL string` field; read `SITE_LOGO_URL` env var |
| `internal/config/config_test.go` | Add tests for `SiteLogoURL` default and env override |
| `internal/api/handlers.go` | Add `LogoURL string` to `PageData`; populate from `s.cfg.SiteLogoURL` in `pageBase()` |
| `templates/base.html` | Render URL logo with fallback to `/logo` binary upload |
| `chart/gameshelf/values.yaml` | Add `siteLogoUrl: ""` |
| `chart/gameshelf/templates/configmap.yaml` | Add `site-logo-url` key |
| `chart/gameshelf/templates/deployment.yaml` | Add `SITE_LOGO_URL` env var from configmap |
| `kots-config.yaml` | Add `site_logo_url` item to `branding` group |
| `helmchart.yaml` | Add `siteLogoUrl` wired from config option |

---

## Task 1: Go backend — config and handlers

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/api/handlers.go`

- [ ] **Step 1: Add failing tests for SiteLogoURL**

In `internal/config/config_test.go`, add two test cases after `TestLoad_Defaults`:

```go
func TestLoad_SiteLogoURL_Default(t *testing.T) {
	t.Setenv("SITE_LOGO_URL", "")
	cfg := Load()
	if cfg.SiteLogoURL != "" {
		t.Errorf("SiteLogoURL default: got %q, want empty string", cfg.SiteLogoURL)
	}
}

func TestLoad_SiteLogoURL_EnvOverride(t *testing.T) {
	t.Setenv("SITE_LOGO_URL", "https://example.com/logo.png")
	cfg := Load()
	if cfg.SiteLogoURL != "https://example.com/logo.png" {
		t.Errorf("SiteLogoURL: got %q, want %q", cfg.SiteLogoURL, "https://example.com/logo.png")
	}
}
```

- [ ] **Step 2: Run tests to confirm they fail**

```bash
cd /Users/adam/programming/GameShelf && go test ./internal/config/... -run TestLoad_SiteLogoURL -v
```

Expected: FAIL — `cfg.SiteLogoURL undefined`

- [ ] **Step 3: Add SiteLogoURL to config.go**

In `internal/config/config.go`, update the `Config` struct to add `SiteLogoURL` after `CustomBrandingEnabled`:

```go
type Config struct {
	DatabaseURL    string
	RedisURL       string
	AdminSecret    string
	Port           string
	SiteName       string
	IdentitySecret string // optional; auto-generated and stored in DB if empty
	SDKServiceURL         string // URL of Replicated SDK sidecar, e.g. http://localhost:3000
	LocalDev              bool   // LOCAL_DEV=true bypasses SDK gates when SDK_SERVICE_URL is unset
	SiteColor             string // default primary color (hex), overridden by DB branding settings
	CustomBrandingEnabled bool   // set by LicenseFieldValue custom_branding_enabled via KOTS
	SiteLogoURL           string // URL of logo image; set via SITE_LOGO_URL env var from KOTS config
}
```

And in `Load()`, add `SiteLogoURL` to the return value (after `CustomBrandingEnabled`):

```go
func Load() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	siteName := os.Getenv("SITE_NAME")
	if siteName == "" {
		siteName = "GameShelf"
	}
	siteColor := os.Getenv("SITE_COLOR")
	if siteColor == "" {
		siteColor = "#3B82F6"
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
		IdentitySecret:        os.Getenv("IDENTITY_SECRET"),
		SDKServiceURL:         os.Getenv("SDK_SERVICE_URL"),
		LocalDev:              os.Getenv("LOCAL_DEV") == "true",
		SiteColor:             siteColor,
		CustomBrandingEnabled: os.Getenv("CUSTOM_BRANDING_ENABLED") == "true",
		SiteLogoURL:           os.Getenv("SITE_LOGO_URL"),
	}
}
```

- [ ] **Step 4: Run tests to confirm they pass**

```bash
cd /Users/adam/programming/GameShelf && go test ./internal/config/... -v
```

Expected: all tests PASS including the two new ones.

- [ ] **Step 5: Add LogoURL to PageData and populate in pageBase()**

In `internal/api/handlers.go`, add `LogoURL string` to the `PageData` struct after `HasLogo`:

```go
type PageData struct {
	SiteName        string
	PrimaryColor    string
	SecondaryColor  string
	BackgroundColor string
	FontFamily      string
	HasLogo         bool
	LogoURL         string
	PageTitle       string
	Token           string // admin token, preserved across form POSTs
	PlayerName      string // pre-filled player name from cookie / identity token
	// SDK-driven banners
	LicenseExpired      bool   // true → red "license expired" banner
	LicenseExpiringSoon bool   // true → yellow "expiring soon" banner (< 30 days)
	LicenseExpiresOn    string // human-readable expiry date for the yellow banner
	UpdateAvailable     bool   // true → blue "update available" banner
	// Page-specific (only one populated per page)
	Games                []db.Game
	Game                 *db.Game
	Scores               []leaderboard.Entry
	DBScores             []db.Score
	AllGames             []db.Game
	Site                 *db.Site
	IdentitySecretMasked  string // shown (masked) on admin panel
	CustomBrandingEnabled bool   // true when custom_branding_enabled license field is "true"
}
```

In `pageBase()`, add `data.LogoURL = s.cfg.SiteLogoURL` after `data.CustomBrandingEnabled = s.cfg.CustomBrandingEnabled`:

```go
	data.CustomBrandingEnabled = s.cfg.CustomBrandingEnabled
	data.LogoURL = s.cfg.SiteLogoURL
```

- [ ] **Step 6: Build to verify no compile errors**

```bash
cd /Users/adam/programming/GameShelf && go build ./...
```

Expected: no output (success).

- [ ] **Step 7: Commit**

```bash
cd /Users/adam/programming/GameShelf && git add internal/config/config.go internal/config/config_test.go internal/api/handlers.go
git commit -m "feat: add SiteLogoURL config field and LogoURL to PageData"
```

---

## Task 2: Update HTML template

**Files:**
- Modify: `templates/base.html`

- [ ] **Step 1: Replace the logo rendering in base.html**

In `templates/base.html`, find line 28:

```html
        {{ if .HasLogo }}<img src="/logo" alt="{{ .SiteName }}" class="h-8 w-auto">{{ end }}
```

Replace it with:

```html
        {{ if .LogoURL }}<img src="{{ .LogoURL }}" alt="{{ .SiteName }}" class="h-8 w-auto">
        {{ else if .HasLogo }}<img src="/logo" alt="{{ .SiteName }}" class="h-8 w-auto">{{ end }}
```

- [ ] **Step 2: Build to verify template parses**

```bash
cd /Users/adam/programming/GameShelf && go build ./...
```

Expected: no output (success). Template parse errors surface at startup — the build compiles the binary but templates are parsed at runtime. No further verification needed at this step.

- [ ] **Step 3: Commit**

```bash
cd /Users/adam/programming/GameShelf && git add templates/base.html
git commit -m "feat: render logo from URL with fallback to binary upload"
```

---

## Task 3: Wire through the Helm chart

**Files:**
- Modify: `chart/gameshelf/values.yaml`
- Modify: `chart/gameshelf/templates/configmap.yaml`
- Modify: `chart/gameshelf/templates/deployment.yaml`

- [ ] **Step 1: Add siteLogoUrl to values.yaml**

In `chart/gameshelf/values.yaml`, add `siteLogoUrl: ""` after `customBrandingEnabled`:

```yaml
siteName: "GameShelf"
adminSecret: "changeme"  # REQUIRED — set a strong secret, e.g. --set adminSecret=... or in a values override
siteColor: "#3B82F6"
customBrandingEnabled: "false"
siteLogoUrl: ""
```

- [ ] **Step 2: Add site-logo-url to configmap.yaml**

In `chart/gameshelf/templates/configmap.yaml`, add `site-logo-url` after `custom-branding-enabled`:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ include "gameshelf.fullname" . }}
  labels:
    {{- include "gameshelf.labels" . | nindent 4 }}
data:
  port: "8080"
  site-name: {{ .Values.siteName | quote }}
  site-color: {{ .Values.siteColor | default "#3B82F6" | quote }}
  custom-branding-enabled: {{ .Values.customBrandingEnabled | quote }}
  site-logo-url: {{ .Values.siteLogoUrl | quote }}
```

- [ ] **Step 3: Add SITE_LOGO_URL env var to deployment.yaml**

In `chart/gameshelf/templates/deployment.yaml`, add the `SITE_LOGO_URL` env var after the `CUSTOM_BRANDING_ENABLED` block (after line 69):

```yaml
            - name: CUSTOM_BRANDING_ENABLED
              valueFrom:
                configMapKeyRef:
                  name: {{ include "gameshelf.fullname" . }}
                  key: custom-branding-enabled
            - name: SITE_LOGO_URL
              valueFrom:
                configMapKeyRef:
                  name: {{ include "gameshelf.fullname" . }}
                  key: site-logo-url
```

- [ ] **Step 4: Run helm lint**

```bash
cd /Users/adam/programming/GameShelf && helm lint chart/gameshelf/
```

Expected: `1 chart(s) linted, 0 chart(s) failed`

- [ ] **Step 5: Commit**

```bash
cd /Users/adam/programming/GameShelf && git add chart/gameshelf/values.yaml chart/gameshelf/templates/configmap.yaml chart/gameshelf/templates/deployment.yaml
git commit -m "feat: add siteLogoUrl through Helm chart values, configmap, and deployment"
```

---

## Task 4: Wire KOTS config and helmchart

**Files:**
- Modify: `kots-config.yaml`
- Modify: `helmchart.yaml`

- [ ] **Step 1: Add site_logo_url to the branding group in kots-config.yaml**

In `kots-config.yaml`, in the `branding` group, add `site_logo_url` after `site_color`. The branding group should look like this after the change:

```yaml
    - name: branding
      title: Branding
      when: '{{repl LicenseFieldValue "custom_branding_enabled" | eq "true"}}'
      items:
        - name: site_color
          title: Primary Color
          type: text
          default: "#3B82F6"
          help_text: "Primary color for the GameShelf UI (hex format, e.g. #3B82F6). Requires the Custom Branding license entitlement."
          validation:
            regex:
              pattern: '^#[0-9A-Fa-f]{6}$'
              message: "Must be a valid hex color code (e.g. #3B82F6)"
        - name: site_logo_url
          title: Logo URL
          type: text
          default: ""
          when: '{{repl LicenseFieldValue "custom_branding_enabled" | eq "true"}}'
          help_text: "URL of your organization's logo image (PNG, SVG, or JPEG). Must be publicly accessible. Leave blank to upload a logo via the admin panel instead."
          validation:
            regex:
              pattern: '^(https?://.*|)$'
              message: "Must be a valid URL starting with http:// or https://, or leave blank."
```

- [ ] **Step 2: Add siteLogoUrl to helmchart.yaml**

In `helmchart.yaml`, add `siteLogoUrl` to the `values:` section after `customBrandingEnabled`:

```yaml
    adminSecret: repl{{ ConfigOption `admin_secret`}}
    siteName: repl{{ ConfigOption `site_name`}}
    siteColor: repl{{ ConfigOption `site_color`}}
    customBrandingEnabled: repl{{ LicenseFieldValue `custom_branding_enabled` }}
    siteLogoUrl: 'repl{{ ConfigOption "site_logo_url" }}'
```

- [ ] **Step 3: Verify both files look correct**

Read `kots-config.yaml` — branding group should now have two items: `site_color` and `site_logo_url`.

Read `helmchart.yaml` — `values:` section should now have `siteLogoUrl` after `customBrandingEnabled`.

- [ ] **Step 4: Commit**

```bash
cd /Users/adam/programming/GameShelf && git add kots-config.yaml helmchart.yaml
git commit -m "feat: add site_logo_url to KOTS config screen and helmchart wiring"
```

---

## Self-Review Checklist

After all tasks complete, run:

```bash
cd /Users/adam/programming/GameShelf && go test ./... && helm lint chart/gameshelf/
```

Expected:
- All Go tests pass
- `1 chart(s) linted, 0 chart(s) failed`
