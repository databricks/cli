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
`cmd/bundle/debug` refschema does. For each scalar field it moves the field through
every ordered pair of a small value set — with `absent` in the set, so adding and
removing a field are just the pairs with `absent` on one side — and records three
things per move: did the plan propose anything, did the apply succeed, and was the
next plan clean.

Everything runs in-process against the direct engine's own `CalculatePlan` and
`Apply`. There is no CLI subprocess and no bundle-file upload: at thousands of
permutations, a `bundle deploy` each would be dominated by sync.

## Verdicts

| verdict | meaning |
| --- | --- |
| `OK` | planned, applied, next plan clean |
| `OK_RECREATE` | same, but the engine replaced the resource |
| `SUPPRESSED` | the planner diffed the field and dropped the change; detail is the engine's own reason |
| `NOT_OBSERVABLE` | the two values are identical in the state sent to the API, so there is nothing to see — an unset bool and an explicit `false` are the usual case |
| `NO_PLAN` | the field diff exists but no action was planned |
| `POST_DEPLOY_DRIFT` | applied, yet the plan taken straight afterwards still wants the same change: deploying never converges |
| `POST_DEPLOY_DRIFT_CHILD` | the field converged but another node of the resource did not; a guard that should stay empty |
| `BACKEND_ERROR` | the API rejected the value — usually the value library needs a valid value for this field |
| `DEPLOY_ERROR` | apply failed for a non-API reason |
| `TIMEOUT` | the operation did not finish inside the per-operation deadline |
| `PLAN_ERROR` | planning failed |
| `UNSETTABLE` | the value could not be written into the config at all |
| `BASE_ERROR` | the transition's starting point would not deploy, so nothing was observed |
| `NOT_IN_STATE` | the field exists in bundle config but not in the state type, so it never reaches the API |

`POST_DEPLOY_DRIFT` and `SUPPRESSED` are where the interesting gaps are. `NOT_OBSERVABLE`
is not a gap; it is the wire format telling the truth.

Two files per resource type: `output/<type>.txt` holds only the results worth looking at,
one line each, so any diff to it is a change in behaviour. `output/<type>.full.txt` holds
every result plus the summary and the not-covered list; it is gitignored, because it moves
whenever a passing row moves.

The suite deliberately does **not** read `resources.yml`. That file is the engine's
answer to many of these cases, so consulting it would just restate the implementation.
The `SUPPRESSED` reason string is the engine explaining itself, which is different.

## Value library

`testdata/fields/<resource_type>.yml` supplies values the generic per-kind defaults cannot
guess — enums, ids, anything the backend constrains:

```yaml
skip:
  name: rename waits out an asynchronous delete
fields:
  compute_size: [MEDIUM, LARGE]
```

Every resource type runs everywhere — local and cloud alike — so most types need no file
at all. A field with no entry gets two values for its Go kind, which is enough to see a
value-to-value move on top of add and remove.

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

On cloud (`CLOUD_ENV` set) only resource types with `cloud: true` run, and the results
go to `output/<resource_type>.<cloud>.txt`, which is not committed: which values a
real backend accepts depends on the workspace.

## Out of scope for now

- remote drift — a change made outside the bundle; this suite only edits config
- `permissions` and `grants`, which are stripped from every config before planning: they
  are separate plan nodes describing an ACL, and leaving them in made every recreate
  report a drifted child against whichever field triggered it. Two configs that differ
  only in those blocks therefore collapse to one, recorded in `output/configs.txt`.
- configs with more than one resource, or with an `-init.sh`
- fields under a slice or map, listed at the end of each report
