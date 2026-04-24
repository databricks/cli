# UCM resources

Reference for every resource kind you can declare in `ucm.yml` as of this
version of the CLI. One section per kind with:

- **Fields** — every field on the struct, required/optional, type.
- **Example** — minimal YAML that deploys cleanly.
- **Engines** — per-engine behavior where it differs.
- **Cross-resource refs** — how the resource interacts with other kinds.

Use `ucm validate` to check a file before running `ucm plan`/`ucm deploy`.

---

## ucm.yml skeleton

Every ucm project starts with the same top-level shape:

```yaml
ucm:
  name: my-deployment           # required; uniquely identifies the deployment
  engine: direct                # optional; "terraform" (default) or "direct".
                                # Override at runtime with DATABRICKS_UCM_ENGINE.

workspace:
  host: https://<workspace>.cloud.databricks.com  # required for workspace-scoped resources
  profile: my-profile           # optional; resolves auth from ~/.databrickscfg

account:                        # optional; only needed for account-scoped verbs
  account_id: <uuid>
  host: https://accounts.cloud.databricks.com

resources:
  catalogs: { ... }
  schemas: { ... }
  grants: { ... }
  storage_credentials: { ... }
  tag_validation_rules: { ... }

targets:                        # optional; per-env overrides merged into the
                                # tree when --target <name> is passed.
  dev:
    default: true
    workspace: { host: https://dev... }
    resources: { ... }
  prod:
    workspace: { host: https://prod... }
    resources: { ... }
```

### Engines

The same `ucm.yml` ships through two interchangeable engines:

| | **terraform** (default) | **direct** |
|---|---|---|
| How | Renders a `main.tf.json`, drives `terraform init` + `terraform plan`/`apply`. | Issues SDK calls directly (`w.Catalogs.Create(...)`, etc.). |
| State | `terraform.tfstate` in the configured backend. | `resources.json` per target. |
| When to pick | Matches DAB workflows; richer plan diff. | No terraform binary needed; fast; fewer moving parts. |

Select with `ucm.engine: direct` in config, `DATABRICKS_UCM_ENGINE=direct`, or leave defaulted.

---

## CLI verbs

The surface today. Flags mirror `databricks bundle` where the semantics transfer; UC-inapplicable DAB flags (`--cluster-id`, `--fail-on-active-runs`, `--verbose`, Git-branch `--force`) are intentionally dropped.

| Verb | Summary | Key flags |
|---|---|---|
| `ucm validate` | Load config, run mutators, emit diagnostics. | `--output text\|json` (default `text`), `--target <name>`, `--include-locations` (hidden) |
| `ucm plan` | Preview changes without mutating state. | `-o text\|json` (`json` emits the structured plan), `--target`, `--force-lock` |
| `ucm deploy` | Apply the plan. | `--target`, `--auto-approve`, `--force-lock` |
| `ucm destroy` | Tear down the target's resources. | `--target`, `--auto-approve`, `--force-lock` |
| `ucm summary` | Print deployed resources with workspace URLs. DAB-style header + per-resource-group sections. | `--target`, `-o text\|json`, `--force-pull` (no-op today), `--include-locations`/`--show-full-config` (hidden, no-op) |
| `ucm schema` | Print the JSON schema for `ucm.yml`. | — |
| `ucm policy-check` | Run validation mutators only (cheap pre-commit target). | `--target` |

Stub verbs (placeholder, not yet implemented): `ucm init`, `ucm generate`, `ucm bind`, `ucm debug`, `ucm diff`, `ucm drift`, `ucm import`.

### --output json

`ucm plan -o json` emits the structured plan:

```json
{
  "plan": {
    "resources.catalogs.sales": { "action": "create" },
    "resources.schemas.raw":    { "action": "update" }
  }
}
```

`ucm validate -o json` emits the full config tree as indented JSON. Useful for IDE integration and programmatic validation.

`ucm summary -o json` emits the config tree too; the text form is DAB-styled for human reading.

### --force-lock

Override an in-progress deploy lock when you know the holder is gone (CI died, SIGKILL'd laptop, etc.). Available on `plan`, `deploy`, and `destroy`. Without it, a second client contending for the same target receives `lock.ErrLockHeld` and the acquisition fails.

---

## Cross-resource references

Every string field accepts two forms:

**Literal (bring-your-own)** — the named object is expected to exist already;
ucm references it read-only.

```yaml
resources:
  catalogs:
    partner:
      name: partner_prod
      storage_root: preexisting_partner_cred   # literal name of a credential
                                               # managed outside this ucm.yml
```

**ucm-managed (`${resources.X.Y.Z}`)** — the referenced object is declared in
the same file. Resolution is engine-specific but transparent to you:

```yaml
resources:
  storage_credentials:
    sales_cred:
      name: sales_cred
      aws_iam_role: { role_arn: arn:aws:iam::111122223333:role/uc-sales }
  catalogs:
    sales:
      name: sales_prod
      storage_root: ${resources.storage_credentials.sales_cred.name}
```

| Engine | How the ref resolves |
|---|---|
| terraform | Rewritten to `${databricks_storage_credential.sales_cred.name}` in the rendered `main.tf.json`. Terraform's own graph orders the dependency. |
| direct | Resolved at load time to the literal value (`sales_cred`). Apply order is declared explicitly in the engine: storage_credentials → catalogs → schemas → grants (reverse on delete). |

Unknown refs (typo'd kind or missing key) fail with a clear error at `validate`/`plan` time.

---

## catalogs

A UC catalog.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Catalog name in UC. |
| `comment` | string | no | Human-readable description. |
| `storage_root` | string | no | Cloud URL or `${resources.storage_credentials.*.name}`/`${resources.external_locations.*.url}` (external_locations land in PR #2). Required for non-metastore-default catalogs. |
| `tags` | map[string]string | no | Validated by `tag_validation_rules`. Inherited by child schemas unless `tag_inherit: false`. |
| `schemas` | map[string]*Schema | no | Nested form (see below). Flattened before any engine sees the tree. |
| `grants` | map[string]*Grant | no | Nested form (see below). |

### Example

```yaml
resources:
  catalogs:
    sales:
      name: sales_prod
      comment: sales domain catalog
      storage_root: ${resources.storage_credentials.sales_cred.name}
      tags:
        cost_center: "1234"
        data_owner: sales
        classification: internal
```

### Engines

- **terraform** → `databricks_catalog.<key>` with `force_destroy=true` and `properties=<tags>`.
- **direct** → `w.Catalogs.Create` / `.Update` / `.Delete`.

### Nested form

Schemas and grants can be declared under a catalog; `FlattenNestedResources`
unrolls them into the top-level maps at load time, injecting the parent
reference. These two forms are equivalent:

```yaml
# Nested
resources:
  catalogs:
    sales:
      name: sales_prod
      schemas:
        raw: { name: raw }
      grants:
        admins:
          principal: sales-admins
          privileges: [USE_CATALOG]
```

```yaml
# Flat
resources:
  catalogs:
    sales: { name: sales_prod }
  schemas:
    raw: { name: raw, catalog: sales }
  grants:
    admins:
      securable: { type: catalog, name: sales }
      principal: sales-admins
      privileges: [USE_CATALOG]
```

Collisions (same key in both flat and nested) are a hard error.

---

## schemas

A UC schema inside a catalog.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Schema name within its catalog. |
| `catalog` | string | yes (flat form) | Parent catalog name. Injected automatically when declared nested. |
| `comment` | string | no | |
| `tags` | map[string]string | no | Validated by `tag_validation_rules`. Merged with parent-catalog tags unless `tag_inherit: false` (schema keys win on conflict). |
| `tag_inherit` | bool pointer | no | Default `true` (nil). Set to `false` to opt out of catalog-tag merging. |
| `grants` | map[string]*Grant | no | Nested form; flattened with `securable = {type: schema, name: <key>}`. |

### Example

```yaml
resources:
  schemas:
    raw:
      name: raw
      catalog: sales
      comment: landing zone
      tag_inherit: false            # don't pull in sales.tags
      tags:
        cost_center: "1234"
        data_owner: sales
        classification: internal
```

### Engines

- **terraform** → `databricks_schema.<key>` with `catalog_name`, `force_destroy=true`, `properties=<tags>`, and `depends_on: [databricks_catalog.<catalog>]` when the parent is ucm-managed.
- **direct** → `w.Schemas.Create` / `.Update` / `.Delete`, applied after the parent catalog.

---

## grants

Assigns UC privileges on a securable to a principal.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `securable.type` | string | yes | `catalog` or `schema`. Other types (storage_credential, external_location, volume, connection) are not yet accepted — they land in later Phase A PRs. |
| `securable.name` | string | yes | UC name of the target object. Can reference a ucm-managed sibling by its key (the converter wires the TF dependency automatically). |
| `principal` | string | yes | User, group, or service principal name (account-level). ucm does not create principals — pass a name that already exists. |
| `privileges` | list[string] | yes | UC privilege names (e.g., `USE_CATALOG`, `USE_SCHEMA`, `SELECT`, `MODIFY`). |

### Example

```yaml
resources:
  grants:
    sales_readers:
      securable: { type: schema, name: raw }
      principal: sales-readers
      privileges: [USE_SCHEMA, SELECT]
```

Or nested under the schema:

```yaml
resources:
  catalogs:
    sales:
      name: sales_prod
      schemas:
        raw:
          name: raw
          grants:
            sales_readers:
              principal: sales-readers
              privileges: [USE_SCHEMA, SELECT]
```

### Engines

- **terraform** → `databricks_grants.<key>`. The `catalog` / `schema` field takes `${databricks_catalog.<k>.name}` or `${databricks_schema.<k>.id}` when the securable is ucm-managed; otherwise the literal name. Emits `depends_on` for ucm-managed parents.
- **direct** → `w.Grants.Update` per securable, reconciled in a single pass after all catalogs/schemas land.

### Grant reconciliation (direct engine)

Grants are not stored per-entry in the remote — they're a set on the
securable. The direct engine reconciles by securable: it reads the current
grants, computes the diff against the declared set, and applies the
combined Add/Remove payload in one call. Removing a grant from `ucm.yml`
unassigns it on the next apply.

---

## storage_credentials

A UC storage credential — the capability UC uses to authenticate to cloud
storage. Exactly one identity shape (AWS / Azure MI / Azure SP / Databricks
GCP SA) must be set.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Credential name in UC. |
| `comment` | string | no | |
| `aws_iam_role` | object | one-of | `{ role_arn: arn:aws:iam::...:role/... }` |
| `azure_managed_identity` | object | one-of | `{ access_connector_id: <ARM id>, managed_identity_id: <ARM id, optional> }` |
| `azure_service_principal` | object | one-of | `{ directory_id, application_id, client_secret }` |
| `databricks_gcp_service_account` | object | one-of | `{}` (empty; presence alone toggles the shape — the GCP SA is managed by Databricks) |
| `read_only` | bool | no | Credential is usable only for read operations. Default `false`. |
| `skip_validation` | bool | no | Skip server-side validation on create. Default `false`. Use sparingly — trades fast-fail for runtime surprises. |

Exactly-one-of on the identity fields is enforced by both the TF converter
and the direct-engine input builder. Missing or multiple identity fields
fail before any API call fires.

### Example (AWS)

```yaml
resources:
  storage_credentials:
    sales_cred:
      name: sales_cred
      comment: sales domain storage access
      aws_iam_role:
        role_arn: arn:aws:iam::111122223333:role/uc-sales
```

### Example (Azure MI)

```yaml
resources:
  storage_credentials:
    shared_cred:
      name: shared_cred
      azure_managed_identity:
        access_connector_id: /subscriptions/s/resourceGroups/rg/providers/Microsoft.Databricks/accessConnectors/uc
```

### Example (Databricks-managed GCP SA)

```yaml
resources:
  storage_credentials:
    data_cred:
      name: data_cred
      databricks_gcp_service_account: {}
```

### Engines

- **terraform** → `databricks_storage_credential.<key>`. Emitted ahead of catalogs in the rendered tree.
- **direct** → `w.StorageCredentials.Create` / `.Update` / `.Delete`. Create runs before any catalog that references it; delete runs after any catalog is torn down.

### Using as a securable for grants

Not yet supported. Today ucm grants only accept `catalog` and `schema`
securable types; grants on storage_credentials / external_locations /
volumes / connections land in a follow-up. You can still manage the
credential itself through ucm and grant access out-of-band.

---

## external_locations

A UC external location. Binds a cloud storage URL to a storage credential
so UC can vend access to tables, volumes, and catalogs placed underneath.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | External location name in UC. |
| `url` | string | yes | Cloud URL (e.g. `s3://…`, `abfss://…`, `gs://…`). |
| `credential_name` | string | yes | Storage credential name. Literal or `${resources.storage_credentials.<key>.name}`. |
| `comment` | string | no | |
| `read_only` | bool | no | Location is usable only for read operations. |
| `skip_validation` | bool | no | Skip server-side validation on create. |
| `fallback` | bool | no | When enabled, fall back to cluster credentials if UC credentials are insufficient. |

Deferred for a follow-up: `encryption_details`, `enable_file_events` / `file_event_queue`.

### Example

```yaml
resources:
  storage_credentials:
    sales_cred:
      name: sales_cred
      aws_iam_role:
        role_arn: arn:aws:iam::111122223333:role/uc-sales
  external_locations:
    sales_loc:
      name: sales_loc
      url: s3://acme-sales/prod
      credential_name: ${resources.storage_credentials.sales_cred.name}
```

### Engines

- **terraform** → `databricks_external_location.<key>`. Emitted after storage_credentials and before catalogs.
- **direct** → `w.ExternalLocations.Create` / `.Update` / `.Delete`. Runs after storage_credentials and before catalogs; reverse on delete.

---

## volumes

A UC volume. Managed (UC provisions storage) or external (user-supplied URL under an external location).

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Volume name within its schema. |
| `catalog_name` | string | yes | Parent catalog. Literal or `${resources.catalogs.<key>.name}`. |
| `schema_name` | string | yes | Parent schema. Literal or `${resources.schemas.<key>.name}`. |
| `volume_type` | string | yes | `MANAGED` or `EXTERNAL` (case-insensitive on input; normalised to upper). |
| `storage_location` | string | cond. | Required for `EXTERNAL`, rejected for `MANAGED`. |
| `comment` | string | no | |

### Example (managed)

```yaml
resources:
  volumes:
    landing:
      name: landing
      catalog_name: sales_prod
      schema_name: raw
      volume_type: MANAGED
      comment: "landing zone"
```

### Example (external)

```yaml
resources:
  volumes:
    archive:
      name: archive
      catalog_name: ${resources.catalogs.sales.name}
      schema_name: ${resources.schemas.raw.name}
      volume_type: EXTERNAL
      storage_location: s3://acme-archive/sales/raw
```

### Engines

- **terraform** → `databricks_volume.<key>`. Registered between schemas and connections.
- **direct** → `w.Volumes.Create` / `.Update` / `.ReadByName` / `.DeleteByName`.

### Known limitation

`UpdateVolumeRequestContent` only accepts `comment`, `new_name`, `owner`. Drift on other fields (`storage_location`, `volume_type`, etc.) is silently dropped by the SDK — the planner currently marks as Update and the wire call is a no-op on those fields. Tracked in issue #62 for a fail-fast or force-recreate follow-up.

---

## connections

A UC foreign-catalog connection — the federation link that lets a foreign catalog reference MySQL / PostgreSQL / Snowflake / etc.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `name` | string | yes | Connection name in UC. |
| `connection_type` | string | yes | e.g. `MYSQL`, `POSTGRESQL`, `SNOWFLAKE`, `REDSHIFT`, `BIGQUERY`. |
| `options` | map[string]string | yes | Connection-specific keys (host, port, user, password, ...). Must be non-empty. Per-type key validation is done by the UC API / terraform provider — ucm just shapes the block. |
| `comment` | string | no | |
| `properties` | map[string]string | no | Arbitrary key/value metadata. |
| `read_only` | bool | no | Connection is read-only. |

### Example

```yaml
resources:
  connections:
    sales_mysql:
      name: sales_mysql
      connection_type: MYSQL
      comment: "foreign sales db"
      options:
        host: mysql.acme.internal
        port: "3306"
        user: uc-reader
      properties:
        purpose: analytics
```

### Engines

- **terraform** → `databricks_connection.<key>`. Registered after volumes.
- **direct** → `w.Connections.Create` / `.Update` / `.GetByName` / `.DeleteByName`.

### Known limitation

`UpdateConnection` only accepts `name`, `new_name`, `options`, `owner`. Drift on `connection_type` / `comment` / `properties` / `read_only` is silently dropped by the SDK. Same follow-up as #62.

---

## tag_validation_rules

Declarative tag policy. Independent of any server-side UC tag feature.

### Fields

| Field | Type | Required | Notes |
|---|---|---|---|
| `securable_types` | list[string] | yes | Which resource kinds this rule applies to. Currently supported: `catalog`, `schema`. |
| `required` | list[string] | no | Tag keys that must be present on every matching securable. |
| `allowed_values` | map[string]list[string] | no | Restricts values for named keys. Keys not listed accept any value. |

### Example

```yaml
resources:
  tag_validation_rules:
    enforce_ownership:
      securable_types: [catalog, schema]
      required:
        - cost_center
        - data_owner
        - classification
      allowed_values:
        classification: [public, internal, confidential, restricted]
```

### When it runs

The `ValidateTags` mutator runs on `validate`, `plan`, and `policy-check`.
Violations produce error-level diagnostics pointing at the offending
securable's YAML span. `deploy` inherits the same check because it runs
`validate` upstream of apply.

---

## Load-time mutators (applied automatically)

These transformations happen before any engine sees the config. You don't
invoke them, but knowing the order explains some of the rules above:

1. **FlattenNestedResources** — lifts nested schemas/grants out of catalogs (and grants out of schemas), injecting parent references. After this step, every resource lives in a top-level flat map.
2. **InheritCatalogTags** — merges a catalog's `tags` into every child schema unless the schema sets `tag_inherit: false`. Schema keys win on collisions.
3. **DefineDefaultTarget** / **SelectDefaultTarget** / **SelectTarget** — picks the active target (from `--target` or `default: true`) and folds its overrides into the top-level tree.
4. **ValidateTags** — runs on `validate`/`plan`/`policy-check` only. Emits diagnostics for missing/disallowed tags.
5. **ResolveVariableReferencesOnlyResources("resources")** (direct engine only) — rewrites `${resources.*}` refs to literal strings for SDK calls. The terraform engine preserves the refs and runs its own `Interpolate` pass later, rewriting to `${databricks_*}` form.

---

## Not yet supported

Deferred:

- Grants on storage_credentials / external_locations / volumes / connections. Today `grants.securable.type` accepts only `catalog` or `schema`.
- `catalog_workspace_binding` (Phase B).
- Account-scoped resources: `metastore`, `metastore_assignment`, `metastore_data_access` (Phase C — requires AccountClient wiring).
- Cloud underlay: S3/ADLS/GCS buckets, IAM, KMS (Phase D).
- Config-level features: `variables:` block + `${var.x}` interpolation (issue #37), `include:` directive (issue #38), expanded validator pack (issues #39, #40).

Check issue #48 (Phase A epic) and #36 (M2 umbrella) for up-to-date status.
