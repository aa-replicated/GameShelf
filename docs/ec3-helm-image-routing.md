# Getting Your Helm Chart to Work with Embedded Cluster v3: Online and Airgap

This guide covers everything you need to configure a Helm chart to work correctly across all three install paths — Helm CLI (CMX), EC3 online, and EC3 airgap — from a single release.

The official docs cover each piece in separate places. This doc assembles the complete picture in one place.

---

## The Three Install Paths

| Path | How it works | Who configures image routing |
|------|-------------|------------------------------|
| **Helm CLI (CMX)** | Customer runs `helm install` directly | Your `values.yaml` defaults |
| **EC3 online** | EC3 installs your chart; nodes have internet access | `helmchart.yaml` `values` section at install time |
| **EC3 airgap** | EC3 installs from a bundle with no internet access | `helmchart.yaml` `values` section + `builder` section at bundle-build time |

`helmchart.yaml` is a KOTS custom resource — it is invisible to `helm install`. It only applies when KOTS/EC3 is the installer.

---

## How EC3 Routes Images

EC3 v3 uses two mechanisms depending on the install type:

**Online:** EC3 configures containerd with registry mirrors pointing to the Replicated proxy (`proxy.replicated.com/proxy/<app-slug>/<original-registry>`). Your chart's image references are rewritten at the `helmchart.yaml` level before Helm sees them.

**Airgap:** EC3 bundles all images into a `.airgap` file at release-build time. At install time it loads them into a local embedded registry (e.g. `10.244.128.11:5000`). Your chart's image references must point to this local registry. EC3 configures containerd mirrors for each registry you declared via `ReplicatedImageRegistry` in `helmchart.yaml`.

---

## The Template Functions

Only a small set of Replicated template functions are available in `helmchart.yaml`. The ones you need for image routing:

### `ReplicatedImageRegistry "registry-host"`

Takes an original registry hostname. Returns:
- **Online:** `proxy.replicated.com/proxy/<app-slug>/<registry-host>`
- **Airgap:** The local embedded registry address (e.g. `10.244.128.11:5000`)

Use this for any image where the registry and repository are separate fields in the chart values (the common case for Bitnami charts and most others).

```yaml
# Online result:  proxy.replicated.com/proxy/myapp/ghcr.io
# Airgap result:  10.244.128.11:5000
registry: 'repl{{ ReplicatedImageRegistry "ghcr.io" }}'
```

**Important:** Pass only the registry hostname — not the full image ref. The repository stays as-is in your chart's values.

### `ReplicatedImageRegistry "registry-host" true`

The `true` flag is `noProxy`. Behavior:
- **Online:** Returns the registry hostname unchanged (no proxy wrapping)
- **Airgap:** Still returns the local embedded registry address

Use this for images that are already hosted at a proxy-aware address and should not be double-wrapped online, but still need local registry routing for airgap.

```yaml
# Online result:  proxy.replicated.com  (unchanged)
# Airgap result:  10.244.128.11:5000
registry: 'repl{{ ReplicatedImageRegistry "proxy.replicated.com" true }}'
```

### `ReplicatedImageName "full-image-ref"`

Returns the complete rewritten image reference (registry + repository + tag). Use this only when your chart has a single combined `image:` field rather than separate `registry:` and `repository:` fields.

---

## The `values` Section

The `values` section in `helmchart.yaml` is merged on top of your `values.yaml` at install time, before Helm renders templates. Template functions are evaluated in the customer's environment.

### Your main app image

Your Helm template constructs the image as `{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag }}`. Override only `registry` — keep `repository` as the bare path (no registry prefix):

```yaml
values:
  image:
    registry: 'repl{{ ReplicatedImageRegistry "ghcr.io" }}'
    repository: 'myorg/myapp'     # bare path, no registry prefix
    pullPolicy: IfNotPresent
```

Do NOT set `repository` to the full proxy URL here. `ReplicatedImageRegistry` returns the registry prefix; Helm concatenates the repository to it.

### Bitnami subcharts (postgresql, redis, etc.)

Bitnami charts split image into `registry` and `repository`. The chart's default `repository` (`bitnami/postgresql`, `bitnami/redis`) is correct — only override `registry`:

```yaml
  postgresql:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
  redis:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
```

Use `index.docker.io`, not `docker.io`. The Replicated proxy distinguishes between them.

### The Replicated SDK subchart

The SDK image lives at `proxy.replicated.com/library/replicated-sdk-image`. It is a special case:

- For **online**, the SDK works without any override — `proxy.replicated.com` is reachable directly.
- For **airgap**, the SDK needs to be redirected to the local registry. Use `noProxy=true` so online installs are unaffected:

```yaml
  replicated:
    image:
      registry: 'repl{{ ReplicatedImageRegistry "proxy.replicated.com" true }}'
```

> **What does NOT work:**
> - `ReplicatedImageRegistry "proxy.replicated.com"` (no noProxy) → online produces a doubled proxy URL (`proxy.replicated.com/proxy/myapp/proxy.replicated.com/...`)
> - `HasLocalRegistry` and `LocalRegistryHost` are **not available** in `helmchart.yaml` — they exist in other KOTS template contexts but not here
> - `ReplicatedImageRegistry (HelmValue ".replicated.image.registry")` — `HelmValue` passes `proxy.replicated.com` as a runtime value but the behavior is unreliable in EC3 beta.1

### `imagePullSecrets` for Bitnami subcharts

Bitnami charts check `global.imagePullSecrets`. Set this in your `values.yaml` so it applies to all subcharts automatically:

```yaml
global:
  imagePullSecrets:
    - name: enterprise-pull-secret
  security:
    allowInsecureImages: true   # required for the local embedded registry (no TLS)
```

`allowInsecureImages: true` is required because the EC3 embedded registry is HTTP, not HTTPS.

---

## The `builder` Section

The `builder` section is used **only by the Replicated Vendor Portal at release-build time** — never at runtime on the customer's machine. Its purpose: render your chart templates with static values so the portal can discover every container image and pull them into the `.airgap` bundle.

Rules:
- No template functions — only static values
- Must include every image that needs to be in the airgap bundle
- Use the **true source registry** for each image (not the proxy URL)

```yaml
  builder:
    image:
      registry: "ghcr.io"
      repository: "myorg/myapp"        # must match your chart's default repository
    replicated:
      image:
        registry: "proxy.replicated.com"
        repository: "library/replicated-sdk-image"
    postgresql:
      image:
        registry: "docker.io"          # Bitnami chart defaults to docker.io
    redis:
      image:
        registry: "docker.io"
```

**If your main app image is missing from `builder`:** The portal falls back to `values.yaml` defaults. If those defaults are set to the full proxy URL (e.g. `proxy.replicated.com/proxy/myapp/ghcr.io/myorg/myapp`) rather than the source registry, the portal may not correctly resolve and bundle the image. The airgap install will then show `not found` against the local registry even though the routing is correct.

**If an image is in `values` but not `builder`:** That image will not be in the airgap bundle. The pod will fail with either `not found` (local registry) or `i/o timeout` (trying to reach the internet).

---

## The `values.yaml` Defaults

`values.yaml` defaults apply to Helm CLI (CMX) installs. Set them to the full proxy URL format so CMX online installs work without any `--set` flags:

```yaml
image:
  registry: proxy.replicated.com
  repository: proxy/myapp/ghcr.io/myorg/myapp
  tag: ""
  pullPolicy: IfNotPresent

postgresql:
  image:
    registry: proxy.replicated.com/proxy/myapp/index.docker.io

redis:
  image:
    registry: proxy.replicated.com/proxy/myapp/index.docker.io
```

The format is `proxy.replicated.com/proxy/<app-slug>/<upstream-registry>`.

---

## The Deployment Template

Your deployment template should construct the image ref as:

```yaml
image: "{{ .Values.image.registry }}/{{ .Values.image.repository }}:{{ .Values.image.tag | default .Chart.AppVersion }}"
```

Do not add conditionals that check `imageProxy.host` or try to build the proxy path in the template. EC3 overrides `image.registry` via `helmchart.yaml` before the template renders — let it do the work.

---

## Complete Working `helmchart.yaml`

```yaml
apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: myapp
spec:
  chart:
    name: myapp
    chartVersion: "0.0.0"   # replaced by CI at release time
  values:
    # --- KOTS config/license values ---
    adminSecret: repl{{ ConfigOption `admin_secret`}}
    siteName: repl{{ ConfigOption `site_name`}}
    customBrandingEnabled: repl{{ LicenseFieldValue `custom_branding_enabled` }}

    # --- Main app image ---
    image:
      registry: 'repl{{ ReplicatedImageRegistry "ghcr.io" }}'
      repository: 'myorg/myapp'
      pullPolicy: IfNotPresent

    # --- Replicated SDK subchart ---
    replicated:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "proxy.replicated.com" true }}'

    # --- Bitnami postgresql subchart ---
    postgresql:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'

    # --- Bitnami redis subchart ---
    redis:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'

  builder:
    image:
      registry: "ghcr.io"
      repository: "myorg/myapp"
    replicated:
      image:
        registry: "proxy.replicated.com"
        repository: "library/replicated-sdk-image"
    postgresql:
      image:
        registry: "docker.io"
    redis:
      image:
        registry: "docker.io"
```

---

## Checklist

Before testing an airgap install:

- [ ] Every image has a `ReplicatedImageRegistry` override in `values` (or is handled automatically by EC3)
- [ ] Every image has a corresponding entry in `builder` with the true source registry
- [ ] `values.yaml` defaults use the full proxy URL format for CMX installs
- [ ] `global.imagePullSecrets` is set in `values.yaml` for subchart image pull auth
- [ ] `global.security.allowInsecureImages: true` is set for the embedded registry
- [ ] The deployment template does not build the proxy URL itself — it uses `image.registry/image.repository:tag` directly
- [ ] A new release has been promoted in the Vendor Portal **after** `builder` changes, so the airgap bundle is rebuilt

---

## Debugging Image Pull Failures

| Error | Likely cause |
|-------|-------------|
| `proxy.replicated.com/proxy/myapp/proxy.replicated.com/...` | `ReplicatedImageRegistry` called without `noProxy=true` on an image already at `proxy.replicated.com` |
| `docker.io/library/replicated-sdk-image` | `ReplicatedImageRegistry` returned empty string; the registry override resolved to nothing |
| `10.x.x.x:5000/myimage: not found` | Routing is correct but image was not bundled — check `builder` section and rebuild airgap bundle |
| `proxy.replicated.com/...: i/o timeout` | Airgap node trying to reach internet — image has no `ReplicatedImageRegistry` override or was not bundled |
| `function "HasLocalRegistry" not defined` | `HasLocalRegistry`/`LocalRegistryHost` are not available in `helmchart.yaml` — use `ReplicatedImageRegistry` instead |
