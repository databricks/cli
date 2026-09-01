# Resource field-support catalog

This suite answers one question for every field a user can set on a bundle resource:
**if I change this field, does the direct engine notice, apply it, and converge?**

The answer for every field lands in `output/<resource_type>.txt`. Those files are
the deliverable — a map of where field support is solid and where it is not. A bad
outcome does not fail the test; a *changed* outcome does, because the goldens are
committed. That makes the suite a regression detector for things like an SDK bump
changing a field's type.

## What it does

For each resource type with a value library in `testdata/fields`, the suite deploys that
library's `base` as a one-resource bundle, then walks the resource's input struct the way
`cmd/bundle/debug` refschema does. For each field it moves the field through every
ordered pair of a small value set — with `absent` in the set, so adding and removing a
field are just the pairs with `absent` on one side — and records three things per move:
did the plan propose anything, did the apply succeed, and was the next plan clean.

The pairs are walked as one chain rather than staged one at a time: the values form a
complete digraph, so a single Eulerian circuit covers every ordered pair exactly once and
each move starts where the last one ended. That halves the deploys, and
`TestTransitionsCoverEveryPairInOneChain` is what guarantees nothing is missed.

A field whose type has no generic value at all — an `any` like `serialized_dashboard` — and a
required field the library gives a single value are reported as not covered rather than
quietly skipped: there is no second value to move to and no absent to move from.

**Slices and maps** are covered two ways. The container is a field in its own right, whose
values are the config's own and that value with one entry dropped — or, for a list of
scalars the config leaves empty, one and two elements of the element's own type — so with `absent`
in the set, one field covers adding and removing an entry as well as adding and removing
the whole container, all with data the backend has already accepted. Separately, a pattern
like `tasks[*].description` is expanded against the deployed config to the indices that
exist, so fields inside an element are tested like any other. A pattern with nothing
behind it in the config is reported as not covered rather than silently passing.

Everything runs in-process against the direct engine's own `CalculatePlan` and
`Apply`. There is no CLI subprocess and no bundle-file upload: at thousands of
permutations, a `bundle deploy` each would be dominated by sync.

Edits are made to the **typed** resource — `libs/structs/structaccess` over
`*resources.Schema` and friends — and synced into the dynamic tree the planner reads the
way any mutator does. That is also what makes "absent" precise: a field is absent when it
holds the zero value and is not in `ForceSendFields`, which is exactly the distinction the
API sees.

## Verdicts

| verdict | meaning |
| --- | --- |
| `OK` | planned, applied, next plan clean |
| `OK_RECREATE` | same, but the engine replaced the resource |
| `SUPPRESSED` | the planner diffed the field and dropped the change; detail is the engine's own reason |
| `NOT_OBSERVABLE` | the two values are identical in the state sent to the API, so there is nothing to see — an unset bool and an explicit `false` are the usual case, as is a field the engine consumes before planning (an alert's `file_path`) |
| `NO_PLAN` | the field diff exists but no action was planned |
| `POST_DEPLOY_DRIFT` | applied, yet the plan taken straight afterwards still wants the same change: deploying never converges |
| `POST_DEPLOY_DRIFT_CHILD` | the field converged but another node of the resource did not; a guard that should stay empty |
| `BACKEND_ERROR` | the API rejected the value — usually the value library needs a valid value for this field |
| `DEPLOY_ERROR` | apply failed for a non-API reason |
| `TIMEOUT` | the operation did not finish inside the per-operation deadline |
| `PLAN_ERROR` | planning failed |
| `UNSETTABLE` | the value could not be written into the config at all |
| `BASE_ERROR` | the transition's starting point would not deploy, so nothing was observed |
| `START_NOT_REACHED` | the starting value deployed without error but the field did not end up holding it, so the move under test could not be set up — a field the API refuses to clear cannot start from `absent`. Retried once on a fresh resource before being recorded |
| `UPDATE_IGNORED` | the apply succeeded and the engine sent the write, but the field's remote value is unchanged on two consecutive reads: the backend accepted the request and ignored this field |
| `STALE_READ` | the write did land, but the read straight after the apply did not show it and the next one did — not a support gap, yet a user planning right after a deploy is shown a change that does not exist |
| `BASELINE_DRIFT` | the field drifts with no config change at all, measured once per resource so it is not blamed on every field tested afterwards |
| `COLLATERAL_DRIFT` | the field under test converged, but updating it left some *other* field drifting; the detail names that field, which is where the fix belongs |
| `OK_INERT` | the resource declares that it ignores local changes to this field, and it does — the claim verified rather than assumed |
| `INERT_NOT_HONOURED` | the resource declares the field inert and the change was applied anyway |
| `SKIPPED` | left out by the value library, with a reason |

`START_NOT_REACHED` and `UPDATE_IGNORED` are two views of the largest class this suite finds:
a field the backend will not clear. The engine sends the write, the backend keeps what it had,
and nothing can then start from `absent`. `SUPPRESSED` with a non-benign reason and
`COLLATERAL_DRIFT` are the next most interesting. `NOT_OBSERVABLE` is not a gap; it is the wire
format telling the truth.

Two files per resource type. `output/<type>.txt` is committed: one line per finding, then
the count of every verdict — including the passing ones, which no line names. Those counts
are what makes a change in *passing* behaviour visible, since a field that starts being
recreated instead of updated moves one `OK` to `OK_RECREATE` and nothing else would show it.

`output/<type>.full.txt` holds every row, the not-covered list, and the evidence indented
under each finding — the post-deploy plan for drift, the whole API error for a rejection. It
is gitignored, because it is full of generated ids and moves whenever any row moves.

The suite reads `resources.yml` for two things only. A field the resource declares an
`output_only` backend output is skipped rather than tested, since a user cannot meaningfully
set one at all. And a field the resource declares it ignores local changes to is tested
anyway, so the claim is checked: it should come back suppressed with exactly that reason,
which is `OK_INERT`, and anything else is `INERT_NOT_HONOURED`.

Nothing else is consulted. `resources.yml` is the engine's answer to most of these cases, so
reading it further would just restate the implementation; the `SUPPRESSED` reason string is
the engine explaining itself, which is different.

## Value library

`testdata/fields/<resource_type>.yml` is the whole fixture for a type: the resource to deploy,
and the values to move its fields through. There is one per type, and it is the only place a
run reads a resource definition from.

`fields` supplies values the generic per-kind defaults cannot guess — enums, ids, anything the
backend constrains. A field naming another object needs a name that exists: a real workspace
rejects the generic `x` outright, so the field would report nothing but `BACKEND_ERROR`.

```yaml
# The resource, rendered into a one-resource databricks.yml and deployed before anything is
# measured. What it declares is what can be tested: a block absent here has no entry for its
# fields to live in, and some blocks only validate as a whole -- a job's git_source needs a
# provider, a url and exactly one ref -- so they cannot be built up one field at a time.
base:
  name: test-job-$UNIQUE_NAME
  git_source:
    git_provider: gitHub
    git_url: https://github.com/databricks/cli.test
    git_commit: abc123

skip:
  git_source.git_branch: mutually exclusive with the seeded git_commit

fields:
  git_source.git_commit: [abc123, def456]
```

A field with no entry gets two values derived from its type: the first two members of an SDK
enum (excluding the `*_UNSPECIFIED` sentinel, which means "unset" and is normalized away), or
two of its Go kind. That is enough to see a value-to-value move on top of add and remove.

Values that a real workspace constrains have to be written down even when the Go type does not
say so — a pipeline's `channel` and `edition` are plain strings the backend validates, and a
cluster size is a t-shirt size. The AWS run is what finds these: locally the fake server takes
anything.

An `OK` that has only ever been seen against the fake server is worth less than one a
workspace has agreed to, and most of all for a field whose value must name something real. The
fake server takes any string, so for such a field `OK` says nothing until it has been checked
against a workspace — see above. That check is impossible for a `local_only` type, whose report
is a record of what the fake server does: `external_locations` has 62 such rows, all in blocks
naming cloud storage the suite does not provision.

A type that cannot run against a real workspace at all declares `local_only: <reason>` and
is skipped on cloud — an external location needs a storage credential with cloud IAM behind it,
and an instance pool cannot be deleted again afterwards.

A service that exists on some clouds and not others is a different case, and declares where it
does: `clouds: [aws]` runs the type on an aws workspace and skips it elsewhere, so a cloud-specific
service is still confirmed somewhere rather than nowhere. This mirrors `CloudEnvs.gcp = false` in
`acceptance/bundle/resources/postgres_*/test.toml`; the blanket exclusions in
`acceptance/bundle/invariant/test.toml` drop those configs from every cloud, including the one
that has the service.

A `skip` key may be a pattern (`aliases[*].id`), matched the way the planner matches its own
field rules, and naming a block skips everything beneath it.

A field that names the resource gets the run's own unique suffix appended to its values, so
two runs against one workspace never ask for the same name — the second would get "already
exists", which says nothing about the engine. Such a field is recognised by `base` templating
its value with `$UNIQUE_NAME`. The suffix is a placeholder until the value is
written, since a rebuilt resource has a new one while the old is still alive, and it is
redacted in the report so the golden is the same on every run.

`base` is the whole resource, and is expanded with the `$VARS` an acceptance config would get —
`$CURRENT_USER_NAME`, `$NODE_TYPE_ID`, `$TEST_DEFAULT_WAREHOUSE_ID` and the rest — so a value can
name the workspace's own user rather than a placeholder only the fake server knows.
`$UNIQUE_NAME` is expanded later, per deploy, because a rebuild gets a new one.

Because `base` becomes the bundle, it can only hold what a bundle can: a field the config format
rejects is rejected here too. That is deliberate — an alert built from a `.dbalert.json` takes
everything but `warehouse_id`, `display_name` and `file_path` from that file, and seeding
`evaluation` in the bundle is a shape the CLI refuses.

`skip` is for a field no single-field edit can exercise — one that only validates as
part of a set (a job's `git_source`), or whose change is correct but ruinously slow (an
app rename waits out an asynchronous delete). Each entry needs a reason, and shows up in
the report as `SKIPPED` so it is not mistaken for coverage.

## Running it

```bash
go test ./bundle/direct/autotest                      # against the testserver
go test ./bundle/direct/autotest -run 'TestFields/schemas' -v
./task test-update-fields                             # regenerate the goldens
```

The local run is part of `./task test`, so it needs no task of its own. The real-workspace runs
do, since they need `CLOUD_ENV` and credentials:

```bash
./task autotest-cloud       # every field; nightly
./task autotest-cloud-pr    # -sample 2; PRs
```

`-sample N` tests N of each type's fields instead of all of them, picked from HEAD so that
successive commits cover different ground and one run's picks follow from its SHA. The reports
stay the same committed goldens: a sampled run is held to the rows belonging to the fields it
picked, plus any row the harness records against itself, which is what catches a type that no
longer deploys at all. The summary counts every field and has no meaningful subset, so it is
not compared -- a change between two *passing* verdicts shows up only in the full run.
`-sample` refuses to run with `-update`, which would truncate the goldens to the sample.

One transition is addressable on its own, and the subtest names carry no shell
metacharacters so no quoting is needed. `-v` prints the post-deploy plan behind any
problem verdict:

```bash
go test ./bundle/direct/autotest -run TestFields/pipelines/.*/dry_run/absent_to_true -v
```

### Checking one field against a real workspace

A whole resource type on cloud is slow (jobs is half an hour) and the interesting question is
usually about a handful of fields. Every field is a subtest, so a few can be driven on their
own:

    CLOUD_ENV=aws go test ./bundle/direct/autotest \
      -run 'TestFields/^apps$/[^/]*/^(budget_policy_id|usage_policy_id)$'

The golden comparison fails on a partial run, which is expected: read the rows in
`output/<type>.<cloud>.full.txt` instead. This is how to check a field the local report calls
`OK` when its value looks like it must name something real, a `*_id` or an ARN or a path. The
fake server takes any string, so `OK` there means nothing until a workspace has agreed: both
`budget_policy_id` and `usage_policy_id` on apps read `OK` locally and are rejected outright on
a real workspace, which names an account-level policy neither has.

On cloud (`CLOUD_ENV` set) the same reports are compared against the same committed
goldens: a divergence means the fake server in `libs/testserver` does not model the API
faithfully, which is worth failing over. The full report goes to
`output/<resource_type>.<cloud>.full.txt` so a cloud run can be read next to a local one.

## Out of scope for now

- remote drift — a change made outside the bundle; this suite only edits config
- `permissions` and `grants`, which are stripped from every fixture before planning: they
  are separate plan nodes describing an ACL, and leaving them in made every recreate
  report a drifted child against whichever field triggered it. They are supported resource
  types in their own right, and `output/configs.txt` leaves them out for the same reason.
- resource types with no value library yet, listed in `output/configs.txt`
- fields under a slice or map, listed at the end of each report

Two divergences from a real workspace are known and left as findings rather than modelled,
because each needs a behavioural change the acceptance suite currently asserts otherwise:

- A SQL warehouse reads back `STARTING`, not `RUNNING`, right after create, so a config asking
  for `started: false` is already satisfied there. Modelling it means the fake server also has
  to move the warehouse to `RUNNING` on a later read, which the engine's waiter depends on.
- A model serving endpoint's `email_notifications` do not take effect from a create, so the field
  drifts from the moment the resource exists — recorded as `BASELINE_DRIFT`, with the plan behind
  it in the full report.

A third, a Genie space's title, is fixed: the fake server named an untitled space `""` where the
backend names it `New Agent`, and the missing default was masking a bug in this suite. That is the
shape these entries usually have, which is why they are worth chasing rather than tolerating.

Two types cannot be driven against the workspace this was verified on, for reasons that are its
capacity rather than the engine's behaviour: a cluster never reaches `RUNNING` because the
instance pool `$TEST_INSTANCE_POOL_ID` names cannot provision instances, and an instance pool
cannot be deleted (which is why that type is `local_only` — every field that recreates hits it).
`clusters` is deliberately *not* marked `local_only`: one workspace's capacity is not a property
of the suite.
