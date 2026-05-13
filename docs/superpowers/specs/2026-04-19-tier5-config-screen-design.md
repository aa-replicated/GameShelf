# Tier 5: Config Screen Design

**Date:** 2026-04-19
**Branch:** demo/tier5
**Status:** Ready for implementation

---

## Goal

Satisfy Tier 5 rubric items 5.0–5.4 by expanding the KOTS config screen with:
- External DB toggle with conditional connection fields (5.0)
- Generated DB password that survives upgrade (5.2)
- Site logo URL as a second configurable feature (5.1, handled in separate spec)
- Regex validation already done on `site_color` (5.3 ✓)
- Help text on all items (5.4 — extend to all new fields)

---

## Current State

`kots-config.yaml` has two groups:
- `gameshelf`: `admin_secret`, `site_name`
- `branding` (license-gated): `site_color` (has regex validation + help_text)

`values.yaml` has `postgresql.enabled: true` and an `externalDatabase` block, but these are not surfaced in the KOTS config screen — the customer can't toggle between embedded/external from the installer UI.

The embedded DB password is hardcoded as `"gameshelf"` in `values.yaml`. No generated password exists.

---

## Design

### 5.0 — External Database Toggle

Add a new `database` config group. A `db_type` dropdown controls whether the embedded or external path is used.

**Config items:**

```yaml
- name: db_type
  title: Database
  type: select_one
  default: embedded
  items:
    - name: embedded
      title: Embedded (recommended)
    - name: external
      title: External PostgreSQL
  help_text: "Choose Embedded to use the built-in PostgreSQL instance, or External to connect to your own PostgreSQL server."

- name: db_host
  title: Database Host
  type: text
  when: '{{repl ConfigOptionEquals "db_type" "external"}}'
  required: true
  help_text: "Hostname or IP address of your PostgreSQL server."

- name: db_port
  title: Database Port
  type: text
  default: "5432"
  when: '{{repl ConfigOptionEquals "db_type" "external"}}'
  help_text: "Port your PostgreSQL server listens on. Default is 5432."
  validation:
    regex:
      pattern: '^\d+$'
      message: "Must be a numeric port number."

- name: db_name
  title: Database Name
  type: text
  default: "gameshelf"
  when: '{{repl ConfigOptionEquals "db_type" "external"}}'
  help_text: "Name of the PostgreSQL database to connect to."

- name: db_user
  title: Database Username
  type: text
  default: "gameshelf"
  when: '{{repl ConfigOptionEquals "db_type" "external"}}'
  help_text: "Username for authenticating with the PostgreSQL database."

- name: db_password
  title: Database Password
  type: password
  when: '{{repl ConfigOptionEquals "db_type" "external"}}'
  required: true
  help_text: "Password for authenticating with the PostgreSQL database."
```

**Helm wiring in `helmchart.yaml`:**

```yaml
postgresql:
  enabled: 'repl{{ConfigOptionEquals "db_type" "embedded"}}'
externalDatabase:
  host: 'repl{{ConfigOption "db_host"}}'
  port: 'repl{{ConfigOption "db_port" | ParseInt}}'
  database: 'repl{{ConfigOption "db_name"}}'
  username: 'repl{{ConfigOption "db_user"}}'
  password: 'repl{{ConfigOption "db_password"}}'
```

---

### 5.2 — Generated Database Password

The embedded DB password must be generated on first install and reused on upgrade. KOTS provides `RandomString` for this — it generates a value once and persists it across upgrades.

Add a hidden config item for the embedded DB password:

```yaml
- name: db_password_generated
  title: Embedded Database Password
  type: password
  hidden: true
  default: '{{repl RandomString 32}}'
  help_text: "Auto-generated password for the embedded PostgreSQL instance. Generated once at install and preserved across upgrades."
```

Wire into `helmchart.yaml`:

```yaml
postgresql:
  auth:
    password: 'repl{{ConfigOption "db_password_generated"}}'
```

`RandomString` generates the value once on first install and KOTS persists it. On upgrade, `ConfigOption "db_password_generated"` returns the same stored value — the DB password never changes.

**Note:** Also need to wire this into the `_helpers.tpl` `databaseUrl` helper so the embedded path uses the generated password rather than `values.yaml`'s hardcoded `"gameshelf"`. The helper currently reads `.Values.postgresql.auth.password` — since `helmchart.yaml` overrides that value, it will flow through correctly.

---

### 5.4 — Help Text

Every config item must have `help_text`. The new items all include it (see above). Existing items:
- `admin_secret` ✓ already has help_text
- `site_name` ✓ already has help_text
- `site_color` ✓ already has help_text

---

## File Changes

| File | Change |
|------|--------|
| `kots-config.yaml` | Add `database` group with `db_type`, `db_host`, `db_port`, `db_name`, `db_user`, `db_password`, `db_password_generated` |
| `helmchart.yaml` | Wire `postgresql.enabled`, `postgresql.auth.password`, and all `externalDatabase.*` fields from config |

No Helm chart template changes required — `_helpers.tpl` already switches between embedded/external based on `postgresql.enabled` and the `externalDatabase.*` values.

---

## Demo Flow (5.0)

1. Install with `db_type = embedded` → show `gameshelf-postgresql-0` pod Running
2. Install with `db_type = external` → show no postgresql pod, app connects to external instance

## Demo Flow (5.2)

1. Install → note the generated password is stored (hidden field)
2. Upgrade to next release → app still connects, no reconfiguration needed

---

## What Is Not Changed

- `chart/gameshelf/templates/_helpers.tpl` — already handles embedded/external switch correctly
- `chart/gameshelf/templates/secret.yaml` — already uses the helper
- Redis — not part of this tier; remains embedded-only
