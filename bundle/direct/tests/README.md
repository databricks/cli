# Resource field-support catalog

This suite answers one question for every field a user can set on a bundle resource:
**if I change this field, does the direct engine notice, apply it, and converge?**

The answer for every field lands in `output/<resource_type>.txt`. Those files are
the deliverable — a map of where field support is solid and where it is not. A bad
outcome does not fail the test; a *changed* outcome does, because the goldens are
committed. That makes the suite a regression detector for things like an SDK bump
changing a field's type.

## What it does

For each config in `acceptance/bundle/invariant/configs` that describes a single
resource, the suite deploys it once, then walks the resource's input struct the way
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

`POST_DEPLOY_DRIFT` and `SUPPRESSED` are where the interesting gaps are. `NOT_OBSERVABLE`
is not a gap; it is the wire format telling the truth.

Two files per resource type. `output/<type>.txt` is committed: one line per finding, then
the count of every verdict — including the passing ones, which no line names. Those counts
are what makes a change in *passing* behaviour visible, since a field that starts being
recreated instead of updated moves one `OK` to `OK_RECREATE` and nothing else would show it.

`output/<type>.full.txt` holds every row, the not-covered list, and the evidence indented
under each finding — the post-deploy plan for drift, the whole API error for a rejection. It
is gitignored, because it is full of generated ids and moves whenever any row moves.

The suite deliberately does **not** read `resources.yml`. That file is the engine's
answer to many of these cases, so consulting it would just restate the implementation.
The `SUPPRESSED` reason string is the engine explaining itself, which is different.

## Value library

`testdata/fields/<resource_type>.yml` supplies values the generic per-kind defaults cannot
guess — enums, ids, anything the backend constrains. A field naming another object needs a
name that exists: a real workspace rejects the generic `x` outright, so the field would
report nothing but `BACKEND_ERROR`.

```yaml
# Seeded into the resource before the first deploy, so this block's fields become
# reachable. Some blocks only validate as a whole -- a job's git_source needs a provider,
# a url and exactly one ref -- and cannot be built up one field at a time from nothing.
# Keep path-valued fields out: base is merged after the mutator pipeline has run.
base:
  git_source:
    git_provider: gitHub
    git_url: https://github.com/databricks/cli.test
    git_commit: abc123

skip:
  git_source.git_branch: mutually exclusive with the seeded git_commit

fields:
  git_source.git_commit: [abc123, def456]
```

Every resource type runs everywhere — local and cloud alike — so most types need no file
at all. A field with no entry gets two values for its Go kind, which is enough to see a
value-to-value move on top of add and remove.

A type that cannot run against a real workspace at all declares `local_only: <reason>` and
is skipped on cloud — an external location needs a storage credential with cloud IAM behind
it, and Lakebase and Postgres resources are not available in every cloud. This mirrors the
per-config cloud exclusions in `acceptance/bundle/invariant/test.toml`.

A `skip` key may be a pattern (`aliases[*].id`), matched the way the planner matches its own
field rules, and naming a block skips everything beneath it.

`base` is expanded with the same `$VARS` the corpus configs use — `$CURRENT_USER_NAME`,
`$UNIQUE_NAME`, `$NODE_TYPE_ID` and the rest — so a seeded value can name the workspace's
own user rather than a placeholder only the fake server knows.

`skip` is for a field no single-field edit can exercise — one that only validates as
part of a set (a job's `git_source`), or whose change is correct but ruinously slow (an
app rename waits out an asynchronous delete). Each entry needs a reason, and shows up in
the report as `SKIPPED` so it is not mistaken for coverage.

## Running it

```bash
go test ./bundle/direct/tests                      # against the testserver
go test ./bundle/direct/tests -run 'TestFields/schemas' -v
./task test-update-fields                                    # regenerate the goldens
```

One transition is addressable on its own, and the subtest names carry no shell
metacharacters so no quoting is needed. `-v` prints the post-deploy plan behind any
problem verdict:

```bash
go test ./bundle/direct/tests -run TestFields/pipelines/.*/dry_run/absent_to_true -v
```

On cloud (`CLOUD_ENV` set) the same reports are compared against the same committed
goldens: a divergence means the fake server in `libs/testserver` does not model the API
faithfully, which is worth failing over. The full report goes to
`output/<resource_type>.<cloud>.full.txt` so a cloud run can be read next to a local one.

## Out of scope for now

- remote drift — a change made outside the bundle; this suite only edits config
- `permissions` and `grants`, which are stripped from every config before planning: they
  are separate plan nodes describing an ACL, and leaving them in made every recreate
  report a drifted child against whichever field triggered it. Two configs that differ
  only in those blocks therefore collapse to one, recorded in `output/configs.txt`.
- configs with more than one resource, or with an `-init.sh`
- fields under a slice or map, listed at the end of each report
