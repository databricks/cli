# Materialization set acceptance tests

These run locally against `libs/testserver/materialization_set.go`, which mirrors
`UpdateMaterializationSet`'s reconcile semantics rather than stubbing them. That is deliberate: one
`output.txt` serves both the local run and a `Cloud = true` run, so every rejection message here is
a claim about the real API, and the cloud run is what checks it. One message is not: the unknown
operation in `OperationsGet`, which the backend resolves inside the entity-store client rather than
from `FeatureStoreMessages`. It is marked as such in the fake and no case reaches it.

## What the update endpoint can do

The set update reconciles membership and nothing else -- `MutableMaterializedFeatureField.All` is
empty, so every member field is write-once. That shapes most of the cases below: a declared change
to any member field is caught by the planner as a `recreate`, and the only change that reaches the
server is a removal.

## What each case pins

| Case | Claim |
|---|---|
| `basic` | create through the long-running operation, then a redeploy that plans nothing -- the server-assigned schedule state and the output-only member fields must not read as drift |
| `bind` | unbinding leaves the set in the workspace and the plan reports `create`, which is the out-of-band shape; binding it back settles the first plan to `skip`, with the destination suppressed exactly as on a bundle-created set |
| `membership` | the delete cascade runs both ways -- removing either member of a feature's pair on its own is refused, while removing a whole pair plans as `update` and the other feature's pair keeps its ids |
| `removal_scope` | the cascade is scoped per feature: where every feature is materialized offline only, each contributes one member and can have it removed alone |
| `schedule_state` | a member the config leaves unset adopts the state the server assigned; a declared change to it is a `recreate`, and that recreate still has to satisfy the create endpoint's rule that a feature's members agree |
| `immutable` | a write-once member field is caught by the planner as `recreate` and never reaches the server; adding a member routes to `update` so the server refuses it by name |
| `destination` | a destination field the server accepts on write and never returns on read is not drift, while a repoint the config actually asked for still recreates -- the two are told apart by whether the persisted value is recoverable |
| `batch` | a batch feature with both destinations creates and settles; repointing the destinations is a `recreate`; pausing the feature is a `recreate` that succeeds when every member agrees and is refused by the create endpoint when they do not |
| `sawtooth` | the offline-and-online batch pair is refused at create, because a set is not expanded into the third materialization; all three declared creates and then plans nothing |
| `streaming` | a lone streaming online member needs no schedule and no cron; declaring a schedule state on it plans as a `recreate` rather than an in-place update |

## Running against a workspace

`batch`, `sawtooth` and `streaming` carry `Cloud = true`. All three reference an existing `Stream`,
because a Stream is not a bundle resource; override `MATSET_CATALOG`, `MATSET_SCHEMA`,
`MATSET_ONLINE_STORE` and `MATSET_STREAM` to move them to another workspace, and set
`MATSET_PROFILE` to pin the bundle to a named profile. **`DATABRICKS_CONFIG_PROFILE` does not reach
the CLI subprocess**, and the harness builds its own workspace client from the ambient environment
before any script runs, so a local cloud run needs working ambient auth to the target workspace.
`batch` also needs `TEST_DEFAULT_WAREHOUSE_ID`, which the harness only supplies off cloud.

The deployment-history leg (`DMS=true`) cannot run **against a workspace**: the bundle deployments
API rejects `features.` and `materialization_sets.` as resource-type prefixes, so every deploy of
either type fails there with `operations.resource_key must have a known resource type prefix`.
Select the `DMS=` leg for a cloud run until that allowlist includes them. The local `DMS=true` leg
does run, because the fake only requires the key to carry a prefix at all.

### Running a cloud leg

Six cases carry `Cloud = true`: `batch`, `sawtooth`, `streaming`, `immutable`, `membership` and
`removal_scope`. `basic`, `bind`, `destination` and `schedule_state` stay local, because every
server-dependent claim they make is already exercised by `batch`.

The cases default to the shared acceptance schema; override `MATSET_CATALOG`, `MATSET_SCHEMA`,
`MATSET_ONLINE_STORE` to move them. These reach the script because the harness forwards ambient
environment to the CLI subprocess, minus the SDK's own auth variables, which it replaces with the
ones it resolved.

```bash
TOK=$(databricks auth token -p <profile> -o json | jq -r .access_token)
DATABRICKS_CONFIG_FILE=/dev/null DATABRICKS_HOST=<workspace host> DATABRICKS_TOKEN="$TOK" CLOUD_ENV=aws TEST_DEFAULT_WAREHOUSE_ID=<warehouse> DATABRICKS_LITESWAP_TARGET=<liteswap unit>   go test . -run 'TestAccept/bundle/resources/materialization_sets/<case>/DMS=$'
```

Four parts of that are load-bearing:

- **`DATABRICKS_CONFIG_FILE=/dev/null`.** With `RecordRequests = true` the harness proxies the run
  and hands the subprocess a config carrying the *resolved config-file path*. The subprocess then
  re-reads it and can fall back to `[__settings__] default_profile`, whose credentials have nothing
  to do with the workspace under test. The symptom is a token refresh against
  `/oidc/accounts/<some other account>/v1/token`.
- **A token rather than a profile.** The harness flattens its resolved auth into environment
  variables, which overrides a `workspace: profile:` pinned in the bundle. A profile whose
  `auth_type` is `databricks-cli` therefore gets re-resolved in the subprocess without the profile.
- **`/DMS=$`.** The `DMS=true` leg cannot run against a workspace: the bundle deployments API
  rejects `features.` and `materialization_sets.` as resource-type prefixes.
- **The unit name only** in `DATABRICKS_LITESWAP_TARGET` -- `liteswap.Apply` prepends
  `testenv://liteswap/`. Passing the full traffic id routes to base staging, where these endpoints
  do not exist.

The workspace needs a declarative feature preview enabled -- `validateAnyDeclarativePreviewEnabled`
rejects every create otherwise, with `declarativePreviewsNotEnabled`.

### Two destination rules the fixtures have to satisfy

Both are enforced on create by `BatchMaterializationUtils`, and both are now mirrored in the fake,
so an invalid fixture fails locally instead of only on cloud.

1. **Every feature in a set must ask for an identical destination *pair*.** The compared value is
   `(offlineStoreConfig, Option[onlineStoreConfig])` per feature, so matching catalog, schema and
   prefix is not enough: a set cannot mix a feature materialized to both stores with one
   materialized offline only. That is why `membership` gives every feature a pair and
   `removal_scope` gives every feature an offline member alone -- the rule makes those two shapes
   mutually exclusive within one set, which is why they cannot be one case.
2. **A feature's offline and online destinations must differ** in catalog, schema or prefix. The
   fixtures suffix the offline prefix with `_off`.

A third thing the fake does not check: that a member's feature exists. Declare the `features`
resources and reference them as `${resources.features.<key>.full_name}`, which also orders the
feature before the set -- a bare name leaves the create racing the feature, which is what made
`sawtooth` flaky.

## Verified against a workspace, with no case of their own

Run through the `daniel-bai-fs2680` swap on 2026-10-19, against a build carrying the write-once
schedule state. Every `Cloud = true` case passed on that run, so their goldens -- including
`batch`'s recreate carrying `PAUSED` on both members -- are confirmed against the real API.

- **Creation across member kinds.** A batch feature with both destinations; two features sharing one
  destination with a pair each; a sawtooth triple; a lone streaming RTM member. The sad paths too:
  the sawtooth pair alone, members of one feature disagreeing on schedule state, two features with
  different destinations, MBM on a non-Kinesis source, and an empty body.
- **Out-of-band creation is caught by the server, not by the planner.** The plan says `create`
  because the bundle has no state for the resource; the deploy then fails with `ALREADY_EXISTS` on
  the feature, and a set whose feature already has materializations fails with
  `RESOURCE_ALREADY_EXISTS`. A set has no name to collide on, so that second check is the only thing
  standing between an unbound set and a duplicate. The `bind` case pins the planner half of this
  locally; the server half needs a workspace.
- **Get redacts the offline destination.** All four members of a two-feature set read back with
  `offline_store_config` absent, which is what the `destination` suppression exists to absorb.

Two behaviours previously recorded here are gone, not untested: a mixed update whose halves are both
legal, and the atomicity of a mixed update when one half is rejected. Both depended on a pause being
a legal update. With no mutable member field left, the only update is a removal, so there is no
mixed update to construct.

## Not covered yet

- **A set's materialized output tables outlive it.** `bundle destroy` removes the set and its
  features, leaving each offline materialization's `*_partial_aggregates` table behind. A cloud case
  that wants a clean schema has to drop those itself.

- **Partial failure.** One member failing does not stop the others, and the error carries what was
  updated, deleted and failed. Reproducing it needs per-member failure injection the fake does not
  have. This matters more than it used to: with field updates gone, a partway failure can only be a
  partway *deletion*, and nothing already deleted is put back.
