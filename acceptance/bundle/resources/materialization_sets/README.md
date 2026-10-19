# Materialization set acceptance tests

These run locally against `libs/testserver/materialization_set.go`, which mirrors
`UpdateMaterializationSet`'s reconcile semantics rather than stubbing them. That is deliberate: one
`output.txt` serves both the local run and a `Cloud = true` run, so every rejection message here is
a claim about the real API, and the cloud run is what checks it.

## What each case pins

| Case | Claim |
|---|---|
| `basic` | create, then a redeploy that plans nothing — the server-assigned schedule state and the output-only member fields must not read as drift |
| `membership` | removing a whole feature's members plans as `update`, not `recreate`; removing one member of a feature on its own is refused because the delete cascades |
| `schedule_state` | pausing the offline member is an in-place update and does not cascade; declaring the state on an online member is refused |
| `immutable` | a write-once member field is caught by the planner as `recreate` and never reaches the server; adding a member routes to `update` so the server refuses it by name |
| `removal_scope` | the delete cascade is scoped per feature: a feature contributing one member can have it removed alone, while removing one member of a feature contributing two is refused |
| `destination` | a destination field the server accepts on write and never returns on read is not drift, while a repoint the config actually asked for still recreates — the two are told apart by whether the persisted value is recoverable |
| `batch` | a batch feature with both destinations creates and settles; repointing the destinations is a `recreate`; an update mixing an acceptable change with an unacceptable one is refused whole, and the acceptable one applies on its own |
| `sawtooth` | the offline-and-online batch pair is refused at create, because a set is not expanded into the third materialization; all three declared creates and then plans nothing |
| `streaming` | a lone streaming online member needs no schedule and no cron, and pausing it is refused |

## Running against a workspace

`batch`, `sawtooth` and `streaming` already carry `Cloud = true`, and all three pass against a workspace. Both reference an existing `Stream`,
because a Stream is not a bundle resource; override `MATSET_CATALOG`, `MATSET_SCHEMA`,
`MATSET_ONLINE_STORE` and `MATSET_STREAM` to move them to another workspace, and set
`MATSET_PROFILE` to pin the bundle to a named profile. **`DATABRICKS_CONFIG_PROFILE` does not reach
the CLI subprocess**, and the harness builds its own workspace client from the ambient environment
before any script runs, so a local cloud run needs working ambient auth to the target workspace. `batch`
also needs `TEST_DEFAULT_WAREHOUSE_ID`, which the harness only supplies off cloud.

The deployment-history leg (`DMS=true`) cannot run against a workspace: the bundle deployments API
rejects `features.` and `materialization_sets.` as resource-type prefixes, so every deploy of either
type fails there with `operations.resource_key must have a known resource type prefix`. Select the
`DMS=` leg until that allowlist includes them.

### The other six cases

Add to `test.toml`:

```toml
Cloud = true
Env.DATABRICKS_LITESWAP_TARGET = "<liteswap unit>"
```

The env var takes the unit name only — `liteswap.Apply` prepends `testenv://liteswap/`. Passing the
full traffic id routes to base staging instead, where these endpoints do not exist.

Four fixture changes are needed first, because the local names are only valid against the fake:

1. **Catalog and schema.** Declare `catalogs` and `schemas` resources using the harness's
   `UNIQUE_NAME` and reference them from the feature names, so parallel runs do not collide.
2. **Online store.** `online_store_name` must name a store that exists on the target workspace.
3. **Features must exist.** Declare the `features` resources the set members reference; the fake
   does not check that a member's feature exists, and the real API does.
4. **One destination for the whole set.** Every feature in a set must name the same
   `catalog_name`, `schema_name` and `table_name_prefix`; differing only in the prefix is rejected
   with `All features must share the same materialization destination.` The fake does not check
   this, so `membership` and `removal_scope` give each feature its own prefix and would both be
   refused as written.

The workspace also needs a declarative feature preview enabled — `validateAnyDeclarativePreviewEnabled`
rejects every create otherwise, with `declarativePreviewsNotEnabled`.

## Verified against a workspace, with no case of their own

These ran against `liteswap-1-4` through the swap but are not committed as cases, because each one
needs a multi-feature set and the shared-destination rule above means that fixture has to be built
differently from the local ones.

- **A mixed update whose halves are both legal is applied whole.** Pausing one feature's offline
  member and removing another feature's members in the same request: both land, the surviving
  members keep their ids, and the paused feature's online member stays `ACTIVE`.
- **A mixed update is atomic when either half is rejected.** Two shapes were tried -- a legal pause
  plus an illegal online pause, and a legal pause plus the illegal removal of one online member --
  and in both the whole request is refused and a read afterwards shows nothing moved, including the
  half that would have been accepted on its own.
- **Out-of-band creation is caught by the server, not by the planner.** The plan says `create`
  because the bundle has no state for the resource; the deploy then fails with `ALREADY_EXISTS` on
  the feature, and a set whose feature already has materializations fails with
  `RESOURCE_ALREADY_EXISTS`. A set has no name to collide on, so that second check is the only thing
  standing between an unbound set and a duplicate.
- **Bind and unbind.** Binding an out-of-band feature and set makes the next plan settle to `skip`
  with no drift; the offline destination is suppressed as `input_only_not_returned` exactly as on a
  bundle-created set. Unbinding leaves the workspace resource in place and the next plan reports
  `create` for it again.

## Not covered yet

- **A set's materialized output tables outlive it.** `bundle destroy` removes the set and its
  features, leaving each offline materialization's `*_partial_aggregates` table behind. A cloud case
  that wants a clean schema has to drop those itself.

- **Partial failure.** One member failing does not stop the others, and the error carries what was
  updated, deleted and failed. Reproducing it needs per-member failure injection the fake does not
  have.
