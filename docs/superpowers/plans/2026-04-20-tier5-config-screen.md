# Tier 5 Config Screen Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an external database toggle with conditional fields and a generated embedded DB password to the KOTS/EC3 config screen, satisfying Tier 5 rubric items 5.0 and 5.2.

**Architecture:** Two files change. `kots-config.yaml` gains a `database` group with a `db_type` dropdown (embedded/external), conditional external connection fields, and a hidden `db_password_generated` item that uses `RandomString 32` to produce a stable password on first install. `helmchart.yaml` wires all new config values to the chart — `postgresql.enabled`, `postgresql.auth.password`, and the full `externalDatabase.*` block.

**Tech Stack:** KOTS Config custom resource (kots.io/v1beta1), KOTS HelmChart custom resource (kots.io/v1beta2), Helm chart `_helpers.tpl` (already handles embedded/external switch — no changes needed)

---

## File Map

| File | Change |
|------|--------|
| `kots-config.yaml` | Add `database` group with 7 new items |
| `helmchart.yaml` | Add `postgresql.enabled`, `postgresql.auth.password`, and `externalDatabase.*` overrides |

---

## Task 1: Add the database config group to kots-config.yaml

**Files:**
- Modify: `kots-config.yaml`

- [ ] **Step 1: Verify current file contents**

```bash
cat kots-config.yaml
```

Expected: 33 lines, two groups (`gameshelf` and `branding`), no `database` group.

- [ ] **Step 2: Add the database group**

In `kots-config.yaml`, add a new group after the closing of the `gameshelf` group (after line 19, before the `branding` group). The full file should look like this:

```yaml
apiVersion: kots.io/v1beta1
kind: Config
metadata:
  name: gameshelf
spec:
  groups:
    - name: gameshelf
      title: GameShelf
      items:
        - name: admin_secret
          title: Admin Password
          type: password
          required: true
          help_text: "Password for the GameShelf admin panel."
        - name: site_name
          title: Site Name
          type: text
          default: "GameShelf"
          help_text: "The name displayed in the browser title and header."
    - name: database
      title: Database
      items:
        - name: db_type
          title: Database
          type: select_one
          default: embedded
          help_text: "Choose Embedded to use the built-in PostgreSQL instance, or External to connect to your own PostgreSQL server."
          items:
            - name: embedded
              title: Embedded (recommended)
            - name: external
              title: External PostgreSQL
        - name: db_password_generated
          title: Embedded Database Password
          type: password
          hidden: true
          default: '{{repl RandomString 32}}'
          help_text: "Auto-generated password for the embedded PostgreSQL instance. Generated once at install and preserved across upgrades."
        - name: db_host
          title: Database Host
          type: text
          when: '{{repl ConfigOption "db_type" | eq "external"}}'
          required: true
          help_text: "Hostname or IP address of your PostgreSQL server (e.g. db.example.com)."
        - name: db_port
          title: Database Port
          type: text
          default: "5432"
          when: '{{repl ConfigOption "db_type" | eq "external"}}'
          help_text: "Port your PostgreSQL server listens on. Default is 5432."
          validation:
            regex:
              pattern: '^\d+$'
              message: "Must be a numeric port number."
        - name: db_name
          title: Database Name
          type: text
          default: "gameshelf"
          when: '{{repl ConfigOption "db_type" | eq "external"}}'
          help_text: "Name of the PostgreSQL database to connect to."
        - name: db_user
          title: Database Username
          type: text
          default: "gameshelf"
          when: '{{repl ConfigOption "db_type" | eq "external"}}'
          help_text: "Username for authenticating with the PostgreSQL database."
        - name: db_password
          title: Database Password
          type: password
          when: '{{repl ConfigOption "db_type" | eq "external"}}'
          required: true
          help_text: "Password for authenticating with the PostgreSQL database."
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
```

- [ ] **Step 3: Verify file looks correct**

```bash
cat kots-config.yaml
```

Expected: ~90 lines, three groups (`gameshelf`, `database`, `branding`).

- [ ] **Step 4: Commit**

```bash
git add kots-config.yaml
git commit -m "feat: add external DB toggle and generated password to config screen"
```

---

## Task 2: Wire database config values into helmchart.yaml

**Files:**
- Modify: `helmchart.yaml`

- [ ] **Step 1: Verify current helmchart.yaml values section**

```bash
cat helmchart.yaml
```

Expected: `values` section has `adminSecret`, `siteName`, `siteColor`, `customBrandingEnabled`, `imageProxy`, `service`, `image`, `replicated`, `postgresql.image`, `redis.image`. No `postgresql.enabled`, no `postgresql.auth`, no `externalDatabase` block.

- [ ] **Step 2: Add database wiring to the values section**

In `helmchart.yaml`, replace the existing `postgresql:` block:

```yaml
    postgresql:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
```

With:

```yaml
    postgresql:
      enabled: 'repl{{ ConfigOption "db_type" | eq "embedded" }}'
      auth:
        password: 'repl{{ ConfigOption "db_password_generated" }}'
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
    externalDatabase:
      host: 'repl{{ ConfigOption "db_host" }}'
      port: 'repl{{ ConfigOption "db_port" }}'
      database: 'repl{{ ConfigOption "db_name" }}'
      username: 'repl{{ ConfigOption "db_user" }}'
      password: 'repl{{ ConfigOption "db_password" }}'
```

- [ ] **Step 3: Verify the full helmchart.yaml looks correct**

```bash
cat helmchart.yaml
```

Expected full file:

```yaml
apiVersion: kots.io/v1beta2
kind: HelmChart
metadata:
  name: gameshelf
spec:
  chart:
    name: gameshelf
    chartVersion: "0.0.0"
  values:
    adminSecret: repl{{ ConfigOption `admin_secret`}}
    siteName: repl{{ ConfigOption `site_name`}}
    siteColor: repl{{ ConfigOption `site_color`}}
    customBrandingEnabled: repl{{ LicenseFieldValue `custom_branding_enabled` }}
    imageProxy:
      host: ""
    service:
      type: NodePort
      nodePort: 30081
    image:
      registry: 'repl{{ ReplicatedImageRegistry "ghcr.io" }}'
      repository: 'aa-replicated/gameshelf'
      pullPolicy: IfNotPresent
    replicated:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "proxy.replicated.com" true }}'
    postgresql:
      enabled: 'repl{{ ConfigOption "db_type" | eq "embedded" }}'
      auth:
        password: 'repl{{ ConfigOption "db_password_generated" }}'
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
    externalDatabase:
      host: 'repl{{ ConfigOption "db_host" }}'
      port: 'repl{{ ConfigOption "db_port" }}'
      database: 'repl{{ ConfigOption "db_name" }}'
      username: 'repl{{ ConfigOption "db_user" }}'
      password: 'repl{{ ConfigOption "db_password" }}'
    redis:
      image:
        registry: 'repl{{ ReplicatedImageRegistry "index.docker.io" }}'
  builder:
    image:
      registry: "ghcr.io"
      repository: "aa-replicated/gameshelf"
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

- [ ] **Step 4: Run helm lint to confirm no structural issues**

```bash
helm lint chart/gameshelf/
```

Expected: `1 chart(s) linted, 0 chart(s) failed`

- [ ] **Step 5: Commit and push**

```bash
git add helmchart.yaml
git commit -m "feat: wire external DB toggle and generated password into helmchart.yaml"
git push
```

---

## Task 3: Verify and deploy

- [ ] **Step 1: Promote a new release in the Vendor Portal**

Push triggers CI which creates a new release. Promote it to the Unstable channel.

- [ ] **Step 2: Install with embedded DB (default)**

During the EC3 install config screen:
- `Database` dropdown: leave as `Embedded (recommended)`
- Complete install

Expected:
- `gameshelf-postgresql-0` pod is Running
- App connects successfully

Verify:
```bash
sudo k0s kubectl get pods -A | grep postgresql
```

Expected: `gameshelf-postgresql-0   1/1   Running`

- [ ] **Step 3: Verify generated password is stable across upgrade**

After install, trigger an upgrade to a newer release (or re-deploy the same release). App should still connect to the DB without any reconfiguration. No `CrashLoopBackOff` or DB connection errors in the gameshelf pod logs:

```bash
sudo k0s kubectl logs -l app.kubernetes.io/name=gameshelf -n <namespace> | grep -i "database\|connect\|error" | tail -20
```

Expected: No connection errors.

- [ ] **Step 4: Install with external DB**

During the EC3 install config screen:
- `Database` dropdown: select `External PostgreSQL`
- Fill in host, port, name, user, password for a real external PostgreSQL instance

Expected:
- No `gameshelf-postgresql-0` pod
- App connects to the external instance

Verify:
```bash
sudo k0s kubectl get pods -A | grep postgresql
```

Expected: no postgresql pod in the output.
