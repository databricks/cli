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

## Running against a workspace

Add to `test.toml`:

```toml
Cloud = true
Env.DATABRICKS_LITESWAP_TARGET = "<liteswap unit>"
```

The env var takes the unit name only — `liteswap.Apply` prepends `testenv://liteswap/`. Passing the
full traffic id routes to base staging instead, where these endpoints do not exist.

Three fixture changes are needed first, because the local names are only valid against the fake:

1. **Catalog and schema.** Declare `catalogs` and `schemas` resources using the harness's
   `UNIQUE_NAME` and reference them from the feature names, so parallel runs do not collide.
2. **Online store.** `online_store_name` must name a store that exists on the target workspace.
3. **Features must exist.** Declare the `features` resources the set members reference; the fake
   does not check that a member's feature exists, and the real API does.

The workspace also needs a declarative feature preview enabled — `validateAnyDeclarativePreviewEnabled`
rejects every create otherwise, with `declarativePreviewsNotEnabled`.

## Not covered yet

- **Sawtooth.** A sawtooth feature in a set must declare all three members itself — batch offline,
  batch online and streaming online — each batch one carrying a `DERIVED` cron schedule with no
  expression, and both online ones the same destination. Declaring only the offline-and-online pair
  is **rejected at create**, not silently expanded: `handleCreateMaterializationSet` stamps a
  materialization set id on every member before delegating, and the normalization that would expand
  the pair is skipped whenever that id is set, deliberately, so that a member maps to exactly one
  materialized feature and the set stays diffable. The rejection names all three materializations.
  A sawtooth fixture needs a `Stream`, which is not a bundle resource; `danielb.fs2680` already has
  usable ones, so the fixture should reference an existing stream rather than create one.
- **Partial failure.** One member failing does not stop the others, and the error carries what was
  updated, deleted and failed. Reproducing it needs per-member failure injection the fake does not
  have.
