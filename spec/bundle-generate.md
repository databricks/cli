# `bundle generate`: bare template, no-bundle mode, full resource coverage

## Goal

1. Let users start from nothing: `bundle init default-bare` creates a minimal bundle, and `bundle generate <type> --init-bare` creates one and writes the generated resource into it.
2. Make `bundle generate` cover the whole bundle resource surface (`bundle/config/resources.go`), one resource family at a time.

An earlier attempt implements the whole plan on branch `janniklasrose/bundle-gen-all` (10 commits, including one-shot commit c51cdb63a). `janniklasrose/bundle-generate-common-implementations` (443390eaf) has a WIP refactor of the generate plumbing. Reuse code from them with `git show <sha>:<path>`; don't merge them.

## Decisions

- No-bundle generate is opt-in (`--init-bare`). Once the UX is proven, consider making it the default when `databricks.yml` is absent.
- Add a dedicated `default-bare` template instead of slimming `default-minimal`, which users already rely on. Upstream has since slimmed `default-minimal` (#5899), so re-check whether `default-bare` is still needed; this branch kept it without writing down why.
- In `--init-bare` mode, reject flags that need an existing bundle: `--bind`, dashboard `--resource`, and dashboard `--watch`. Dashboard generate's bundle-specific features must not silently degrade.

## Plan (one reviewable commit or PR each)

1. Bare template in `libs/template/templates/default-bare`: `databricks.yml`, `resources/`, `src/`, `.gitignore`, and a short README. Register it in `libs/template/template.go` and in the `bundle init` help.
2. `--init-bare` for every generate subcommand: materialize the bare template, then write the YAML and files into its `resources/` and `src/`.
3. Shared generate plumbing in `cmd/bundle/generate/`: key derivation, file naming, saver setup, bundle vs. auth-only context, and bind gating. Keep per-type conversion in `bundle/generate/`, and don't build a generic exporter until two or three families show a stable pattern. Add a parity test that fails when a resource type in `bundle/config/resources.go` has neither a generate command nor an explicit exclusion.
4. Resource families, each revertible on its own:
   - UC core: `catalogs`, `schemas`, `external_locations`, `volumes`. Add bind only where the type already supports `Exists` and binding.
   - MLflow and serving: `models`, `experiments`, `model_serving_endpoints`. Keep UC `registered_models` clearly separate from MLflow `models` in command names and help.
   - Compute and SQL: `sql_warehouses`, `clusters`. Expect extra normalization of server-managed fields, and prove that the generated YAML deploys without drift.
   - UC advanced: `registered_models`, and `quality_monitors` in its own commit because of API churn.
   - Secrets and Lakebase: `secret_scopes` exports only scope metadata and ACLs, never secret values, and the help must say so. Add `database_instances`, `synced_database_tables`, `postgres_projects`, `postgres_branches` and `postgres_endpoints` if their read APIs are stable. Feature-gate `database_catalogs` if needed.
5. Help snapshots in `acceptance/bundle/help`, a support matrix (which types support bind and which need an existing bundle), and a `.nextchanges/bundles/` fragment.

Testing: acceptance tests first (`acceptance/bundle/generate`, `acceptance/bundle/templates`), plus unit tests for non-trivial conversion helpers.

## Status on this branch

Done: steps 1 and 2, and part of 3.

- 2725b5d0a, cc235a356, 9ae6a8305: `default-bare` template, registered in `bundle init`, with template test `acceptance/bundle/templates/default-bare/minimal`.
- 82852caff, 514482b44, 32634e15d: `--init-bare` on the job, pipeline, dashboard, alert, app and genie_space generators via `cmd/bundle/generate/generate_helper.go` (`ensureGenerateBundle`, `initializeBareBundle`, `validateFlagsForInitBare`). It rejects `--bind`, `--resource` and `--watch`.
- 98d367e03: `cmd/bundle/generate/parity_test.go`.
- cbbf98483: regenerated help output.
- f369a8dc6: `.nextchanges/bundles/bundle-generate-init-bare.md`.
- 650a6e8f8: fixes a "no cmdIO found in the context" panic in about 28 generate acceptance tests. It drops generate's PersistentPreRunE and makes `root.MustWorkspaceClient` skip `logdiag.InitContext` when logdiag is already set up. That changes shared code in `cmd/root/auth.go` and needs a careful review.

Verified locally: acceptance `bundle/generate` and `bundle/templates` pass.

Open:

- The parity test doesn't read `bundle/config/resources.go`. It compares hand-written maps, so a new resource type doesn't fail it; derive the list from the `Resources` struct instead.
  - It lists `vector_search_endpoints` as supported, but there is no such generate command.
  - `parity_test.go` isn't gofmt-formatted.
  - After merging main it also needs `features` (#6952).
- Acceptance coverage is only `generate/job-init-bare-with-bind-fails`. Missing: a successful `--init-bare` run, and the `--resource` and `--watch` rejections.
- 2725b5d0a also refactors `bundle init` flag handling into `resolverFromInitFlags`. Nothing else uses it, so split it out or drop it. The original plan wanted to fix swapped `--tag`/`--branch` bindings there, but that bug no longer exists on main.
- Step 4 (resource families) and the support matrix from step 5 have not been started.

## Risks

- `quality_monitors`, `database_catalogs` and some Lakebase APIs may need feature gating if SDK or backend support is incomplete.
- `secret_scopes` can't be exported with full fidelity; the docs and help must say so plainly.
