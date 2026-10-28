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

## Create, update and delete are all long-running

Each write verb returns a `databricks.longrunning.Operation` and the adapter waits on it, so the set
a create or update produces is read off the operation rather than the response body; a delete's
operation settles with no response at all. Execution is synchronous server-side, so the operation is
already terminal when the call returns and the wait settles on its first poll.

Each verb also takes a `request_id`, and the server reserves an operation record against it before
acting: a repeat carrying a token already used gets the first operation back and changes nothing.
The adapter mints one per call, so the SDK's own retry of a lost response resolves to the first
operation instead of acting twice. The engine's `retryOnTransient` calls the adapter afresh and so
mints a new token; that is safe here because it never retries a create without the adapter opting
in, and re-running an update or a delete reconciles to the same state. No case replays a token,
because every adapter call mints its own.

## A failure after the operation is reserved is not an HTTP error

The backend raises only while it can still reject the request outright, and that boundary sits
earlier than "every validation". On the create path only three rules are reached before the record
is reserved: a member list that is empty, a supplied set id that already exists, and the schedule
states agreeing, which a wire-level validation hook checks before the handler runs at all. The
batch-and-streaming mixing rule, both destination rules and the sawtooth triple rule all live inside
the batch create the handler calls *after* reserving, so each of them settles the record and comes
back as HTTP 200. Measured against the live build, not read off the handler: `batch` and `sawtooth`
each failed on a golden still expecting a 400. Past that point a failure is the operation's outcome,
so all three verbs settle the record FAILED and return it with HTTP 200, carrying the error code and
message on the operation and the per-member outcome on the metadata's progress arm. A creation has
no progress arm: it has no per-member outcome to report.

The generated wait flattens that into one string and reads no metadata, so the adapter re-reads the
settled operation to recover the code structurally and appends the members the change left alone.
It returns an `APIError` so the engine classifies and renders the failure as it would any other,
with the 200 the call actually returned as the status -- which maps to no sentinel, deliberately: a
member that could not be torn down must not reach the destroy step's permission-denied leniency,
whose purpose is to forgive a resource that is already gone.

One gap this leaves is the engine's, not ours, and `settled_failure` pins it: `DeploymentUnit.Delete`
forgives a delete error matching none of the three cases it knows, so a partly deleted set exits
zero with its state dropped. The deployment-history leg is the one path that records and returns it.

## What each case pins

| Case | Claim |
|---|---|
| `basic` | create through the long-running operation, then a redeploy that plans nothing -- the server-assigned schedule state and the output-only member fields must not read as drift; the destroy's operation round trip |
| `bind` | unbinding leaves the set in the workspace and the plan reports `create`, which is the out-of-band shape; binding it back settles the first plan to `skip`, with the destination suppressed exactly as on a bundle-created set |
| `membership` | the delete cascade runs both ways -- removing either member of a feature's pair on its own is refused, while removing a whole pair plans as `update` and the other feature's pair keeps its ids |
| `removal_scope` | the cascade is scoped per feature: where every feature is materialized offline only, each contributes one member and can have it removed alone; the update's operation round trip |
| `schedule_state` | a member the config leaves unset adopts the state the server assigned; a declared change to it is a `recreate`, and that recreate still has to satisfy the create endpoint's rule that a feature's members agree |
| `immutable` | a write-once member field is caught by the planner as `recreate` and never reaches the server; adding a member routes to `update` so the server refuses it by name; destroy then redeploy is the route those refusals leave open, and the re-created set plans clean |
| `destination` | a destination field the server accepts on write and never returns on read is not drift, while a repoint the config actually asked for still recreates -- the two are told apart by whether the persisted value is recoverable |
| `batch` | a batch feature with both destinations creates and settles; neither member of its pair can be removed alone, and removing both empties the set, which the update endpoint refuses in favour of delete; repointing the destinations is a `recreate`; pausing the feature is a `recreate` that succeeds when every member agrees and is refused by the create endpoint when they do not; a set mixing a batch-path feature with a streaming-path one is refused at create |
| `sawtooth` | the offline-and-online batch pair is refused at create, because a set is not expanded into the third materialization; all three declared creates and then plans nothing; the three members are one cascade group, so each of the three strict subsets of them is refused -- with the message depending on whether the removal names the member a delete is applied to -- and removing all three empties the set, which the update endpoint refuses |
| `settled_failure` | a failure the server hits after reserving an operation comes back as HTTP 200 with the operation FAILED: the adapter recovers the error code from the operation and the per-member outcome from its metadata, the update fails naming each member left behind and writes nothing, and the destroy reports success anyway because the engine forgives a delete error it cannot classify |
| `shorthand` | a feature's whole materialization declared once in `sawtooth_materialized_features` expands into its three members, and in `batch_offline_and_online_materialized_features` into its two; neither authoring field survives into the summary, a redeploy plans nothing, and one edit to the online destination reaches both online members while the offline one stays put. A batch schedule the service refuses is reported by the service naming the feature, not by the CLI -- the expansion judges nothing it is given. One arm mixes a shorthand entry with members written out in full, so the expansion is appended rather than substituted; another carries the batch pair shorthand alone, with no member written out. The last declares both authoring fields in one set, which the service refuses because a sawtooth entry has to carry a DERIVED schedule and a batch pair a MANUAL one, and one request cannot mix cron schedule modes. Its fixtures write the feature windows three ways: seconds only, extended units alone (`3d`), and units mixed (`2d1h35m40s`) |
| `invalid_members` | two declarations reaching one member are refused while the config is being read, with no request made: two members differing only outside the member key, and a shorthand entry colliding with a member written out. The message is the service's own wording for the same rejection, and both offenders are named |
| `streaming` | a streaming online member needs no schedule and no cron; declaring a schedule state on it, or repointing its destination, plans as a `recreate` rather than an in-place update; with two features materialized to one destination and no offline member between them, each online member is its own cascade group, so one can be removed alone as an `update` |

## Running against a workspace

`batch`, `sawtooth`, `streaming` and `shorthand` reference an existing `Stream`, because a Stream
is not a bundle resource. **The `MATSET_STREAM` default names one developer's Stream and goes stale
when it is deleted**, which fails the run at the feature create with `Stream '...' not found`; list
what the workspace has with `GET /api/2.0/feature-engineering/streams?parent=<catalog>.<schema>` and
override. Override `MATSET_CATALOG`, `MATSET_SCHEMA`,
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

Three cases carry `Cloud = true`: `immutable`, `membership` and `removal_scope`.

`batch`, `sawtooth`, `streaming` and `shorthand` are local only, because each needs a Stream source
and a Stream is neither a bundle resource nor a CLI command -- so the integration suite has no way
to provision one, and it would also need a Unity Catalog connection to a Kafka topic whose payload
the feature definitions name. Run those four against a workspace by hand, with the overrides below.
`basic`, `bind`, `destination` and `schedule_state` stay local because every server-dependent claim
they make is already exercised by `batch`.

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

## Cloud runs 2026-10-20

Run against the `daniel-bai-fs2680` swap on workspace 2978662807315490. Two swap builds, and the
difference between them is the point.

**First build, universe `afc23c250043d`.** `batch`, `sawtooth`, `streaming` and `immutable` passed;
`membership` and `removal_scope` failed, both refused with:

```
The fields 'pipeline_schedule_state' of materialized feature '<id>' of feature '<name>' cannot be
changed. Only removing members is supported; recreate the materialization set instead.
```

`comparableMaterializationSetMembers` copied the requested `pipelineScheduleState` raw onto the
comparison baseline while normalising the trigger beside it, so a bundle -- which never declares
the field, it being a backend default -- compared `None` against the stored `ACTIVE` and had every
membership change refused. Isolated live: the same removal succeeded when the surviving member
carried `pipeline_schedule_state: ACTIVE` and failed when it did not, nothing else different.

**Second build, universe `8438b2287fb03`.** Fixed, and the validations moved into
`MaterializationSetUtils`. An omitted state now falls back to the stored one
(`.orElse(normalizedExistingMaterializedFeature.pipelineScheduleState)`), and the changed-field list
is derived from `MaterializedFeature.scalaDescriptor.fields` rather than a hand-maintained `Seq`, so
a member field added later is covered without touching this check.

All six cases have passed against that build. The one golden this cost was `membership`: the cascade
refusal was reworded, and the fake carried the old text.

```
old: Changing one materialized feature of feature 'F' also changes '<id>', which the request does
     not change. Apply the same change to every materialized feature of the feature.
new: Changing materialized features '<ids>' of feature 'F' also changes materialized features
     '<ids>', which the request does not change. Apply the same change to the unrequested
     materialized features as well.
```

The new message names the removed members of the offending feature, so the fake groups removed
members by feature and sorts the ids the way `FeatureStoreMessages.quotedIds` renders them.
`materializedFeatureFieldCannotBeChanged` was reworded to match the server too, though no case
reaches it -- the planner classifies every member field as a recreate, so a declared change never
becomes an update request.

**The workspace throws intermittent 500s that are not this resource.** Across five runs of the
`DMS=` leg, individual cases failed on `POST /api/2.0/feature-engineering/features` with `Unable to
create Feature '<name>' in UC. If the issue persists, please retry or contact support.`, and once on
`Failed to create Lakebase table`. `immutable` lost three runs in a row this way, then passed twice in
a row when run on its own -- so the trigger is the parallel load of six bundles deploying at once,
not the fixture. Its two features are identical but for their names, and which one fails varies; a
direct two-feature create reproduced nothing, serial or concurrent. Run a case on its own before
reading a red run as a regression.

**Third build, universe `275193b7facc8`.** The deletion cascade refusal became two messages,
picked by whether the removal names the member a delete is applied to:
`materializationSetMembersCannotBeRemovedAlone` when it does not, and
`deletionAffectsUnrequestedMaterializedFeatureMembers` when it does. The fake branches the same way
and `membership`'s two arms now show one each. A set mixing a batch-path feature with a
streaming-path one is also refused, which the fake had not modelled; `batch`'s last arm pins it,
with the message captured from this build.

All six `Cloud = true` cases pass against this build, run one at a time, with no golden changed --
so every message in the removal matrix is the server's own text and not a reading of its source.
That covers both cascade messages, the empty-set refusal that sends a caller to the delete endpoint,
the batch-versus-streaming mixing refusal, and the `streaming` case's claim that two stream features
materialized to one destination can share a set, which until then rested on the shared-destination
rule comparing a `(offline, Option[online])` pair that is identical for both.

**Fourth build, universe `cd7a68312ca3`** -- the promotion tip, carrying the settled-failure
contract. All six `Cloud = true` cases pass against it, run one at a time. Three messages had been
reworded (`materializationSetMembersCannotBeRemovedAlone`, and both partial-change messages, which
now count what got through and point at the operation's progress), and two refusals moved from an
HTTP 400 to a settled FAILED operation. The fake and four goldens were corrected from what the live
run returned.

**Fifth build, universe `92bd718e722a0`** -- the tip of `stack/fs2680-update-set-lro`, run through
the `daniel-bai-fs2680` swap on 2026-10-23. All seven `Cloud = true` cases pass with no golden
changed, run one at a time, as does `features/lifecycle` on its `DMS=` leg. Two of them had claims
nothing live had settled: `shorthand` had never run against a workspace at all, and `batch` carried
two arms -- the null member-list spelling and the drop-the-resource delete route -- that had only
run against the fake. So the emptied-set refusal's "Delete the set instead" is the service's own
text, and the one-segment `materialized_features` path classifies as an update on the live API and
not just in the unit tests.

`shorthand` is the case that run first justified: a sawtooth entry creates the three members the
service wants, a redeploy plans nothing, one edit to the online destination recreates exactly the
two online members, a shorthand entry and members written out in full coexist in one set, and the
DERIVED-schedule refusal its first arm asserts is the service's own text.

The swap's own log is what proves the routing: during the 63-second run it served 5
`CreateMaterializationSetRequest`, 7 `DeleteMaterializationSetRequest`, 6
`GetMaterializationSetRequest`, 15 `ListMaterializationSetMembersRequest`, 3
`CreateMaterializationSetMembersRequest` and 4 `DeleteMaterializedFeatureRequest`, interleaved with
this run's own fixture names. Zero `UpdateMaterializationSetRequest`, which is the case's one
coverage limit: every membership edit it makes is to an immutable field, so it plans a recreate and
never reaches the update endpoint. `batch`'s emptied-member arms are what exercise that.

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

- **A real member-teardown failure.** `settled_failure` produces the settled-FAILED shape
  from a marker the fake looks for in a feature name, because a real teardown fails on
  infrastructure no request can ask for. The adapter's reading of that shape is pinned; that
  the server emits it for a genuine failure is not, and cannot be from here.

- **A settled-but-failed operation.** Every refusal these cases reach is an HTTP error on the call
  itself, raised before the operation record exists, so the wait is never what fails. The branch
  that reports a failed operation -- naming the set from the operation metadata -- is covered only
  by `TestMaterializationSetOperationTarget`, and the same injection the partial-failure case needs
  would cover it end to end.
