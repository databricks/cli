# NEXT CHANGELOG

## Release v0.299.1

### CLI

* `databricks api` now works against unified hosts. Adds `--account` to scope a call to the account API and `--workspace-id` to override the workspace routing identifier per call. A `?o=<workspace-id>` query parameter on the path (the SPOG URL convention used by the Databricks UI) is also recognized as a per-call workspace override, so URLs pasted from the browser route correctly.
* JSON output for single objects now uses standard `"key": "value"` spacing (matching list output and `encoding/json` defaults).

### Bundles
* Validate that resource keys do not contain variable references ([#5169](https://github.com/databricks/cli/pull/5169))
* engine/direct: Drop the deployment state entry on a recreate before the follow-up `Create`, so a `Create` failure no longer leaves a broken state with `invalid state: empty id` on the next `bundle plan` ([#5173](https://github.com/databricks/cli/pull/5173)).
* `bundle debug list-targets`: skip nil entries in the targets map instead of panicking when a target is declared with a null value ([#5203](https://github.com/databricks/cli/pull/5203)).

### Dependency updates

* Added `github.com/jackc/pgx/v5` v5.9.1 (MIT) as a new dependency. Used by an experimental Postgres command added in this release; the package is dormant for users who do not invoke that command.
