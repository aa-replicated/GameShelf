# Tier 5: Site Logo URL Feature Design

**Date:** 2026-04-19
**Branch:** demo/tier5
**Status:** Ready for implementation

---

## Goal

Add a `SITE_LOGO_URL` environment variable that allows a logo to be set via URL rather than file upload. This satisfies Tier 5 item 5.1 (second configurable app feature wired through config screen) alongside the existing `site_color` feature.

The URL approach is simpler than file upload for the config screen path — no binary storage, no upload endpoint, just a string passed as an env var.

---

## Current State

The app already has a full file-upload logo path:
- `POST /admin/logo` stores binary in `sites.logo_data` (BYTEA)
- `GET /logo` serves it from the DB
- `base.html` renders `<img src="/logo">` when `HasLogo` is true
- The `sites` table already has a `logo_url` TEXT column (from `001_schema.sql`) — unused

The file upload path is gated behind `CustomBrandingEnabled`. This spec adds a parallel URL path, also gated behind the same entitlement.

---

## Design

### Priority: URL over uploaded file

When `SITE_LOGO_URL` is set, use it. When the URL is empty, fall back to the uploaded binary logo (existing behavior). This means:

- `LogoURL` field added to `PageData`
- Template uses `LogoURL` if set, falls back to `/logo` endpoint if `HasLogo` is true
- No existing behavior changes

### Data flow

```
SITE_LOGO_URL env var
  → config.Config.SiteLogoURL
    → pageBase() → PageData.LogoURL
      → base.html: <img src="{{ .LogoURL }}">
```

The existing `logo_url` DB column is NOT used for this feature — the URL comes from the env var, set at install time via the config screen. This keeps it simple and consistent with how `SITE_NAME` and `SITE_COLOR` work.

---

## Changes

### 1. `internal/config/config.go`

Add `SiteLogoURL string` to Config struct. Read `SITE_LOGO_URL` env var with empty string default.

### 2. `internal/api/handlers.go`

Add `LogoURL string` to PageData struct.

In `pageBase()`, set `data.LogoURL`:
- If DB site exists and `site.LogoURL` is non-empty: use it
- Else if `s.cfg.SiteLogoURL` is non-empty: use it
- Else: leave empty (HasLogo path handles binary upload)

### 3. `templates/base.html`

Replace:
```html
{{ if .HasLogo }}<img src="/logo" alt="{{ .SiteName }}" class="h-8 w-auto">{{ end }}
```

With:
```html
{{ if .LogoURL }}<img src="{{ .LogoURL }}" alt="{{ .SiteName }}" class="h-8 w-auto">
{{ else if .HasLogo }}<img src="/logo" alt="{{ .SiteName }}" class="h-8 w-auto">{{ end }}
```

### 4. `kots-config.yaml`

Add `site_logo_url` to the `branding` group (already license-gated):

```yaml
- name: site_logo_url
  title: Logo URL
  type: text
  default: ""
  help_text: "URL of your organization's logo image (PNG, SVG, or JPEG). Must be publicly accessible. Leave blank to upload a logo via the admin panel instead."
  validation:
    regex:
      pattern: '^(https?://.*|)$'
      message: "Must be a valid URL starting with http:// or https://, or leave blank."
```

### 5. `helmchart.yaml`

Add to the `values` section:

```yaml
siteLogoUrl: 'repl{{ConfigOption "site_logo_url"}}'
```

### 6. `chart/gameshelf/values.yaml`

Add to the top-level values:

```yaml
siteLogoUrl: ""
```

### 7. `chart/gameshelf/templates/deployment.yaml`

Add env var to the container spec:

```yaml
- name: SITE_LOGO_URL
  valueFrom:
    configMapKeyRef:
      name: {{ include "gameshelf.fullname" . }}
      key: site-logo-url
```

### 8. `chart/gameshelf/templates/configmap.yaml`

Add:

```yaml
site-logo-url: {{ .Values.siteLogoUrl | quote }}
```

---

## File Changes Summary

| File | Change |
|------|--------|
| `internal/config/config.go` | Add `SiteLogoURL` field, read `SITE_LOGO_URL` env var |
| `internal/api/handlers.go` | Add `LogoURL` to PageData, populate in pageBase() |
| `templates/base.html` | Render URL logo with fallback to binary logo |
| `kots-config.yaml` | Add `site_logo_url` item to branding group |
| `helmchart.yaml` | Wire `siteLogoUrl` from config |
| `chart/gameshelf/values.yaml` | Add `siteLogoUrl: ""` default |
| `chart/gameshelf/templates/configmap.yaml` | Add `site-logo-url` key |
| `chart/gameshelf/templates/deployment.yaml` | Add `SITE_LOGO_URL` env var |

---

## What Is Not Changed

- File upload path (`POST /admin/logo`, `GET /logo`) — unchanged, still works
- `sites.logo_url` DB column — not used by this feature; URL comes from env var only
- `internal/db/queries.go` — no changes needed
- `internal/api/admin.go` — no changes needed

---

## Demo Flow (5.1)

1. Install with `site_logo_url` = a publicly accessible image URL
2. Show logo appearing in the app header
3. Go to config, clear the URL, apply
4. Show logo gone from the header
