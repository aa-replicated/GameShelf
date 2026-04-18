# EC3 Image Registry Fix Design

**Date:** 2026-04-17
**Branch:** demo/tier4
**Status:** Approved for implementation

---

## Problem

Pods fail to pull images on EC3 v3 (online and airgap). The gameshelf, postgresql, and redis pods
get `ErrImagePull` with a doubled proxy URL:

```
proxy.replicated.com/proxy/gameshelf/proxy.replicated.com/aa-replicated/gameshelf:0.1.75
```

The replicated SDK pod works because its chart already uses the correct `proxy.replicated.com`
registry format natively.

### Root cause

Three bugs across recent commits:

1. **`ReplicatedImageRegistry` called without full image ref context** — passing just the registry
   host (e.g., `"ghcr.io"`) returns only the proxy host (`proxy.replicated.com`). Passing the full
   image ref (e.g., `"ghcr.io/aa-replicated/gameshelf"`) returns the full proxy prefix
   (`proxy.replicated.com/proxy/gameshelf/ghcr.io`). The working directory has already partially
   applied this fix (unstaged).

2. **`imageProxy.host` not cleared in `helmchart.yaml`** — the deployment template has an
   `imageProxy.host` branch that prepends `proxy.replicated.com/proxy/gameshelf/` in front of
   `image.registry`. Since `image.registry` is set to the result of `ReplicatedImageRegistry`
   (already a proxy prefix), the result doubles the proxy host.

3. **`ReplicatedImageRepository` calls present** — `ReplicatedImageRepository` is not a
   documented EC3 v3 function. Its return value is unpredictable and overrides the chart's correct
   default repository values. All `ReplicatedImageRepository` overrides must be removed.

---

## How EC3 v3 Image Routing Works

EC3 v3 uses two template functions in `helmchart.yaml` (not in Helm templates):

- **`ReplicatedImageRegistry "<full-image-ref>"`** — for charts with separate `registry` and
  `repository` fields. Given a full image ref as context, returns the registry prefix to use:
  - Online: `proxy.replicated.com/proxy/<app-slug>/<upstream-registry>`
  - Airgap: `<local-embedded-registry-address>`

- **`ReplicatedImageName "<full-image-ref>"`** — for charts with a single image field. Returns
  the complete rewritten image reference.

EC3 v3 automatically configures cluster-level authentication and containerd registry mirrors. No
`imagePullSecrets` injection is needed in the chart for EC3 installs.

---

## Changes

### 1. `helmchart.yaml`

**Clear `imageProxy.host`** so the deployment template uses the `else` branch
(`image.registry/image.repository:tag`) instead of re-wrapping the registry in the proxy path.

**Fix `ReplicatedImageRegistry` inputs** — pass the full image ref for each image, not just the
registry host.

```yaml
values:
  imageProxy:
    host: ""                          # clear so deployment template uses image.registry directly
  image:
    registry: 'repl{{ ReplicatedImageRegistry "ghcr.io/aa-replicated/gameshelf" }}'
    # no repository override — chart default "proxy/gameshelf/ghcr.io/aa-replicated/gameshelf" is correct
    pullPolicy: IfNotPresent
  replicated:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "proxy.replicated.com/library/replicated-sdk-image" }}'
      # no repository override
  postgresql:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "index.docker.io/bitnami/postgresql" }}'
      # no repository override — Bitnami default "bitnami/postgresql" is correct
    volumePermissions:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io/bitnami/os-shell" }}'
        # no repository override
  redis:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "index.docker.io/bitnami/redis" }}'
      # no repository override — Bitnami default "bitnami/redis" is correct
```

All existing `ReplicatedImageRepository` overrides are removed. They override the chart's own
correct repository values with an undocumented function.

Expected resolved values:

| Image | Online | Airgap |
|-------|--------|--------|
| gameshelf | `proxy.replicated.com/proxy/gameshelf/ghcr.io` | `<local-registry>` |
| postgresql | `proxy.replicated.com/proxy/gameshelf/index.docker.io` | `<local-registry>` |
| redis | `proxy.replicated.com/proxy/gameshelf/index.docker.io` | `<local-registry>` |
| os-shell (pg init) | `proxy.replicated.com/proxy/gameshelf/index.docker.io` | `<local-registry>` |
| replicated SDK | `proxy.replicated.com` (unchanged) | `<local-registry>` |

### 2. `deployment.yaml`

Remove the `imageProxy.host` branch. Always build the image ref as
`image.registry/image.repository:tag`. The `imageProxy.host` conditional exists to support CMX
(Helm CLI) installs; for EC3 installs, `helmchart.yaml` clears `imageProxy.host`, so the else
branch already handles EC3 correctly. Remove the dead branch to simplify.

**Before:**
```yaml
{{- if .Values.imageProxy.host }}
image: "{{ .Values.imageProxy.host }}/proxy/{{ .Values.imageProxy.appSlug }}/{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
{{- else }}
image: "{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
{{- end }}
```

**After:**
```yaml
image: "{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
```

CMX (Helm CLI) installs: `values.yaml` keeps `imageProxy.host: proxy.replicated.com` as a
documented value for reference, but the template no longer reads it. CMX customers pulling through
the proxy set `image.registry` directly (e.g., `--set image.registry=proxy.replicated.com/proxy/gameshelf/ghcr.io`).

> **Note:** This is a behavior change for CMX installs. Any existing CMX install instructions that
> rely on `imageProxy.host` will need to be updated to set `image.registry` and `image.repository`
> to the full proxy-prefixed values instead.

### 3. `values.yaml`

Update the `image` block defaults to match the proxy URL format so CMX online installs work
out of the box without needing extra `--set` flags:

```yaml
image:
  registry: proxy.replicated.com
  repository: proxy/gameshelf/ghcr.io/aa-replicated/gameshelf
  tag: ""
  pullPolicy: Always
```

The `imageProxy` block can be removed or left as documentation-only (it is no longer read by any
template).

`postgresql.image.registry` and `redis.image.registry` in `values.yaml` are already correct
(`proxy.replicated.com/proxy/gameshelf/index.docker.io`) — no changes needed.

---

## What Is Not Changed

- `chart/gameshelf/Chart.yaml` — no changes
- Bitnami subchart `values.yaml` defaults — postgresql and redis registry values are already correct
- Replicated SDK subchart — the SDK owns its own image values; the `helmchart.yaml` override
  handles airgap rewriting
- `imagePullSecrets` — EC3 v3 handles cluster-level auth automatically; no changes needed

---

## Verification

After deploying a new release:

1. All 5 pods reach `Running`: gameshelf, postgresql, redis, replicated SDK, and
   postgresql-volumePermissions init container completes
2. Online EC3: image refs resolve through `proxy.replicated.com/proxy/gameshelf/...`
3. Airgap EC3: image refs resolve through the local embedded registry
4. CMX (Helm CLI) online: `helm install` with default values pulls images correctly
