---
description: How to add or extend bundle deploy telemetry metrics
globs:
  - "libs/telemetry/**/*.go"
  - "bundle/phases/telemetry.go"
  - "bundle/phases/resources_metadata.go"
  - "bundle/metrics/**/*.go"
paths:
  - "libs/telemetry/**/*.go"
  - "bundle/phases/telemetry.go"
  - "bundle/phases/resources_metadata.go"
  - "bundle/metrics/**/*.go"
---

# Extending bundle deploy telemetry

The deploy payload is declared twice: the proto in the universe repo (`proto/logs/frontend/databricks_cli/bundle_deploy.proto`, source of truth for the backend table) and the Go struct in `libs/telemetry/protos/` that the CLI serializes to JSON. Ingestion ignores unknown fields, so the two PRs can land in either order — a field the CLI sends before the proto lands is dropped, not rejected.

## Adding a new resource type

**RULE: A new bundle resource type needs no telemetry change.** Per-type counts come from `resources_metadata.resources[*]`, which `resourceMetadata` in `bundle/phases/resources_metadata.go` derives from `Resources.AllResources()` — exhaustive by construction and guarded by `TestResourcesAllResourcesCompleteness`.

**RULE: Do not add new `resource_<type>_count` fields to `BundleDeployEvent`.** The 12 that exist, and the aggregate `resource_count`, are `deprecated = true` in the universe proto, kept only for continuity of existing dashboards. Adding more re-creates what `resources_metadata` fixed: a hand-maintained field list that silently lags the resource types, making a new type indistinguishable from zero adoption. New dashboard panels should read `resources_metadata`.

## Adding a new metric

1. Add the field to the relevant message in `libs/telemetry/protos/`, and populate it in `bundle/phases/telemetry.go`.
2. Refresh the acceptance golden: `go test ./acceptance -run '^TestAccept/bundle/telemetry' -update`. Anything engine-dependent belongs in a per-engine `out.*.$DATABRICKS_BUNDLE_ENGINE.txt` file rather than the engine-agnostic `out.telemetry.txt`; see the comments in that test's `script`.
3. Send the universe PR adding the same field to the proto, with a `compliance.data_label` annotation matching the neighbouring fields.

**RULE: Never send a resource name, path, or any other user-authored string.** Names are PII. Resource identity goes through the numeric/UUID ID lists (`resource_job_ids` and friends), and only for types that have such an ID.

**RULE: Do not add `omitempty` to a count or measurement that can legitimately be zero.** An explicit `0` distinguishes "nothing of this kind in this deploy" from "CLI too old to report the field"; with `omitempty` both arrive as null and the population can't be sized.

Other things worth knowing:

- A one-off boolean needs no proto change: add a key constant to `bundle/metrics/metrics.go` (snake_case, matching the constant name) and call `b.Metrics.SetBoolValue(...)`. It lands in `experimental.bool_values`.
- Metrics that may be removed later belong in `BundleDeployExperimental`, which carries no compatibility guarantees.
- Positional encodings such as the `upload_file_sizes` histogram are frozen once the metric has adoption: changing a bucket bound silently changes the meaning of every previously reported entry.
- Anything measured from the deployment state is engine-dependent. The direct engine keeps per-resource state in `resources.json` and reports sizes; the terraform path leaves `StateSizeBytes` zero and reports only counts.

## Where the data lands

Deploy events land in `main.team_eng_deco.bundle_deploy_telemetry`; the dashboard is `app/dabs_telemetry` in `databricks-eng/ds-projects`. The daily pipeline reads `entry.bundle_deploy_event.*` wholesale, so a new field arrives on its own, but it has to be added to a dashboard panel by hand to become visible.
