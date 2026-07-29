---
description: How to add or extend bundle deploy telemetry metrics
globs: libs/telemetry/**/*.go
paths:
  - "libs/telemetry/**/*.go"
  - "bundle/phases/telemetry.go"
  - "bundle/metrics/**/*.go"
---

# Extending bundle deploy telemetry

The deploy payload is declared in two places: the proto in the universe repo (source of truth for the backend table) and the Go struct in `libs/telemetry/protos/` that the CLI serializes to JSON. Ingestion ignores unknown fields, so the universe PR and the CLI PR are independent and can land in either order — a field the CLI sends before the proto lands is dropped, not rejected.

**RULE: Every resource type in `bundle/config.Resources` has a `resource_<singular>_count` field in `protos.BundleDeployEvent`.** A type without one reports zero adoption forever, and nothing surfaces the omission except someone reading the dashboard and concluding the type is unused. `TestSetResourceCountsCoversAllResourceTypes` in `bundle/phases` fails when a resource type is added without a count.

## Adding a count for a new resource type

1. Add the field to `BundleDeployEvent` in `libs/telemetry/protos/bundle_deploy.go`. The JSON name is `resource_` + the singular of the bundle config key + `_count` (`sql_warehouses` → `resource_sql_warehouse_count`). Keep the field order the same as `config.Resources` so the golden files read in a predictable order.
2. Add one assignment to `setResourceCounts` in `bundle/phases/telemetry.go`.
3. Run `go test ./bundle/phases -run TestSetResourceCountsCoversAllResourceTypes`.
4. Refresh the acceptance golden: `go test ./acceptance -run '^TestAccept/bundle/telemetry' -update`. Count fields have no `omitempty`, so each new one shows up as an explicit `0` in `acceptance/bundle/telemetry/deploy/out.telemetry.txt`.
5. Send the universe PR adding the same field to the bundle deploy proto.

**RULE: Do not add `omitempty` to count fields.** An explicit `0` distinguishes "deployed a bundle with none of this type" from "CLI too old to report the field"; with `omitempty` both arrive as null and the population can't be sized.

**RULE: Never send a resource name, path, or any other user-authored string in telemetry.** Names are PII. Resource identity is reported through the numeric/UUID ID lists (`resource_job_ids` and friends), and only for types that have one — see the comment on those fields.

## Other kinds of metrics

- A one-off boolean needs no proto change: add a key constant to `bundle/metrics/metrics.go` (snake_case, matching the constant name) and call `b.Metrics.SetBoolValue(...)`. It lands in `experimental.bool_values`.
- Metrics that may be removed later belong in `BundleDeployExperimental`, which carries no compatibility guarantees.
- Positional encodings (for example the `upload_file_sizes` histogram) are frozen once the metric has adoption: changing a bucket bound silently changes the meaning of every previously reported entry.
- `resources_metadata` is populated only by the direct engine, because it is derived from that engine's `resources.json` state. Terraform deploys omit it, so counts from it are not comparable to the `resource_*_count` fields.

## Where the data lands

Deploy events land in `main.team_eng_deco.bundle_deploy_telemetry`; the dashboard is `app/dabs_telemetry` in `databricks-eng/ds-projects`. A new field is picked up by the daily pipeline run because it reads `entry.bundle_deploy_event.*` wholesale, but it has to be added to a dashboard panel by hand to be visible.
