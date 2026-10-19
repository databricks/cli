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
| `membership` | the delete cascade is per feature and runs both ways -- removing either member of a feature's pair on its own is refused, a feature contributing one member can have it removed alone, and removing a whole pair plans as `update` with the surviving members keeping their ids |
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

### The other cases

`basic`, `bind`, `destination`, `immutable`, `membership` and `schedule_state` are local only. Two of
them are planner assertions a workspace cannot strengthen -- `bind` and `destination` both pin a skip
reason the planner derives with no server involvement -- so converting those buys little. The other
four do make claims about the server.

To convert one, add to `test.toml`:

```toml
Cloud = true
Env.DATABRICKS_LITESWAP_TARGET = "<liteswap unit>"
```

The env var takes the unit name only -- `liteswap.Apply` prepends `testenv://liteswap/`. Passing the
full traffic id routes to base staging instead, where these endpoints do not exist.

Three fixture changes are needed first, because the local names are only valid against the fake:

1. **Catalog and schema.** Declare `catalogs` and `schemas` resources using the harness's
   `UNIQUE_NAME` and reference them from the feature names, so parallel runs do not collide.
2. **Online store.** `online_store_name` must name a store that exists on the target workspace.
3. **Features must exist.** Declare the `features` resources the set members reference, and
   reference them as `${resources.features.<key>.full_name}` so the engine orders the feature before
   the set. The fake does not check that a member's feature exists and the real API does; a bare name
   also leaves the create racing the feature, which is what made `sawtooth` flaky.

A fourth rule used to bite and no longer does: every feature in a set must name the same
`catalog_name`, `schema_name` and `table_name_prefix`, or the create is refused with
`All features must share the same materialization destination.` The fake does not check this, so the
multi-feature fixtures declare one shared destination already.

The workspace also needs a declarative feature preview enabled -- `validateAnyDeclarativePreviewEnabled`
rejects every create otherwise, with `declarativePreviewsNotEnabled`.

## Verified against a workspace, with no case of their own

Run through the `daniel-bai-fs2680` swap on 2026-10-19, against a build carrying the write-once
schedule state.

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

- **A recreate carrying `PAUSED` on every member.** The two server rules it rests on were checked
  live -- agreeing members create, disagreeing ones are refused -- but not `PAUSED` on create itself.
  This is the one claim in `batch`'s golden a workspace has not confirmed.

- **A set's materialized output tables outlive it.** `bundle destroy` removes the set and its
  features, leaving each offline materialization's `*_partial_aggregates` table behind. A cloud case
  that wants a clean schema has to drop those itself.

- **Partial failure.** One member failing does not stop the others, and the error carries what was
  updated, deleted and failed. Reproducing it needs per-member failure injection the fake does not
  have. This matters more than it used to: with field updates gone, a partway failure can only be a
  partway *deletion*, and nothing already deleted is put back.
