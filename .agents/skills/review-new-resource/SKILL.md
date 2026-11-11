---
name: review-new-resource
description: Review a branch or PR that adds a new bundle resource type to the direct engine, as a maintainer would. Checks scope, API shape and plan semantics first, then adapter contract, wiring completeness, fake server scope, test coverage and whether the tests actually ran, generated-file hygiene and user-facing text. Use when asked to review a change that adds a resources.<plural> type, or when a review of such a PR should start from the design rather than from line comments.
---

# Review a new resource PR

A new resource should be one PR, about 250 to 400 hand-written Go lines, an adapter of 60 to 130
lines with no plan overrides, a fake of 70 to 100 lines, and three to five acceptance directories.
Deviations are where review time goes. Judge the change against the repo's precedents and the API
readiness gate, not against the author's design; a gap in the API is fixed in the API, not in the
resource and never in the engine.

Read `../author-new-resource/references/api-readiness.md`, `wiring.md`, `testing.md` and
`pitfalls.md` once; they are the standards this review applies.

## 1. Collect the change

```bash
TARGET=HEAD                       # or the commit you were given
BASE=$(git merge-base "$TARGET" origin/main)
git diff --stat "$BASE" "$TARGET"
git diff "$BASE" "$TARGET" -- . ':!*.generated.yml' ':!bundle/schema/jsonschema.json' ':!python/' ':!acceptance/**/output.txt' ':!acceptance/**/out.*'
```

Read every changed non-generated file in full at `TARGET` (`git show "$TARGET":path`). Skim
generated files only to confirm they follow from the source. Do not read existing review comments
before forming your view; read the PR description for intent if you have it.

Identify the resource: its plural, the config type under `bundle/config/resources/`, the adapter
under `bundle/direct/dresources/`, its `configs/<plural>.yml` and `.generated.yml`, the fake under
`libs/testserver/`, and the acceptance directories.

## 2. Triage before line comments

Compute and write down:

```bash
git diff --numstat "$BASE" "$TARGET" -- '*.go' ':!*_test.go' ':!*generated*' ':!python/' | awk '{a+=$1} END {print a}'
git show "$TARGET":bundle/direct/dresources/<singular>.go | wc -l
git show "$TARGET":libs/testserver/<plural>.go | wc -l
git ls-tree --name-only "$TARGET" acceptance/bundle/resources/<plural>/
```

Then compare with the two most recent resource additions on `origin/main`. A number more than twice
the precedent is the first comment, asked as a question about the cause, before anything else.

## 3. Required passes

Every pass produces its table for the worksheet, even when empty. A finding needs code you read;
drop anything you cannot point to.

### P1. Scope

List every changed file that is not part of adding this resource: another resource's behaviour,
generic mutators, engine code under `bundle/direct/*.go` or `libs/structs/`, messages for other
types, bundle-wide syntax features. Each is a split request, made once, before reviewing its
content.
Table: `file | belongs to the new resource? | if not, what it changes`

### P2. API shape and plan semantics

Apply the readiness gate to the result. For each row, cite the file and line.

- Identity: how the engine id is formed; `provided_id_fields` or `updatable_id_fields` present
  and matching; a composed id has a composer and a parser on the config type.
- `KeyedSlices`: every key function returns a real field of the element. A key built from several
  fields or from a value the client does not send is a redesign comment, pointing at the gate.
- `OverrideChangeDesc`: present? For each branch, which yml rule it stands in for and why that rule
  could not be written. Default position: it should not exist in a new resource.
- `configs/<plural>.yml`: every hand-written rule has a comment naming the API behaviour; rules
  that mirror `IMMUTABLE`, `OUTPUT_ONLY` or `INPUT_ONLY` are bridges, so the comment also names
  the annotation the proto lacks. Whether the upstream request was filed is a question to the
  author, not something the description has to state.
- `configs/<plural>.generated.yml` and `apitypes.generated.yml`: the type picked is the resource
  type, not a request type; input-only and output-only fields match the API docs.
- `DoUpdate`: present only if the API can update what the config can change; a removal-only or
  partial update is a design comment. For a `*` mask, find the evidence that this API accepts it:
  the API doc line in `.codegen/cli.json`, or a cloud test that performs an update. No evidence is
  a question, not a pass.
- Every ignore rule, hand-written or generated: what does ignoring this field cost the user, and
  how do they restore intended state if it drifts? An `ignore_remote_changes` on a field the user
  sets, or an `ignore_local_changes` anywhere, needs that answer in a comment.
- Sibling resources: if another resource manages the same kind of object or the same service
  (an older API generation, a related type in the same package), list where this one behaves
  differently (approvals, presets, URL, fix-ups) and ask whether each difference is intended.
- Long-running calls: id returned before polling, polling in the wait hooks; inline waits carry a
  comment that explains why the hook does not fit.
Table: `item | finding (file:line) | verdict: fine / ask upstream / redesign at API`

### P3. Adapter contract

Check against `bundle/direct/dresources/README.md` and the `IResource` interface in `adapter.go`:
one SDK call per `DoCreate`/`DoUpdate`/`DoDelete`; SDK errors returned unwrapped; no nil checks on
arguments the framework guarantees; `DoCreate` cannot return an empty id; `RemapState` is a trivial copy; remapping lives in `DoRead`;
`Exists` uses `apierr.IsMissing`; `GetURL` returns `"", false` with a comment only when there is no
UI page; marshal methods on the config type have the right receivers.
Table: `method | contract point | ok? (file:line)`

### P4. Wiring completeness

Walk the checklist in `../author-new-resource/references/wiring.md`. For each row: present, absent
with a justifying comment, or absent. Pay attention to the rows that nothing enforces: state load
fixture, invariant matrix registration, deployment-history allowlist or exclusion, deploy and
destroy approval groups together, URL plus its command tests, UC dependency capture when the
resource names a catalog or schema, run_as and presets decisions.
Table: `checklist row | present? (file:line) | if absent: justified?`

### P5. Fake server

Size against precedent; scope limited to create, read, update, delete and server-assigned fields;
JSON error bodies with `error_code`; no failure triggers in fixture data; a pending operation path
if the API is asynchronous; redaction only of fields the proto marks input-only. Every modelled
backend rule beyond CRUD needs either a server source reference or a `Cloud = true` test that
exercises it.
Table: `behaviour modelled | needed by the CLI? | evidence (server ref or cloud test)`

### P6. Tests and whether they ran

- Canonical set present: `basic`, `lifecycle` with `Cloud = true`, `remote-delete`,
  `deployment/bind/<singular>`, invariant template, PyDABs support dir when eligible.
- Invariant template registered in `acceptance/bundle/invariant/test.toml`; migrate and
  auto-migrate exclusions added.
- `out.test.toml` files show the expected `Cloud` and `DMS` values; a `DMS=true` exclusion has a
  comment.
- The recreate is executed, not only planned; `print_requests.py` shows the delete and create.
- Scripts: `trace`, one-line `title`, `musterr`, `print_requests.py`, `replace_ids.py`, projection
  of API output; no inline `jq` on the request log; no `[[Server]]` stub for a route the fake models.
- Goldens: no unresolved `${resources...}`; no full API dumps. A golden that shows a failed
  operation reported as success, an object left behind after destroy, or an error swallowed is a
  finding even when the cause is pre-existing engine behaviour: the test must not enshrine it.
- Deployment history: if the `DMS=true` leg is excluded anywhere, the type is absent from
  `dmsResourceTypes` in `libs/testserver/bundledeployments.go`; ask whether the backend allowlist
  change is tracked and say so in the review body. An exclusion without that is a comment.
- Unit tests: `testConfig` entry populates every field; completeness-list tests updated rather than
  special-cased (`run_as_test.go`, `apply_bundle_permissions_test.go`, `workspace_open_test.go`,
  `all_test.go`). A branch on the resource name inside a shared harness is a comment.
- Cloud coverage: the `lifecycle` directory (or the directory that plays its role) has
  `Cloud = true` in its `out.test.toml`. The PR's Integration Tests check runs every changed
  `Cloud = true` test, so that check is the evidence; do not ask for a run link, an environment, or
  a statement about the run.
- Bundle-wide behaviour the resource opted into (a dev-mode preset such as a prefix or a paused
  schedule, run_as rejection, UC dependency capture, approval prompts) has an acceptance test that
  shows it; a unit test alone is a comment.
- Any claim in this review that a test exists, or that a file does something, cites `path:line`
  you read. Verify counts with a command before stating them.
Table: `expected test | present? | ran? (out.test.toml / CI check) | note`

### P7. Generated files and descriptions

`annotations.yml` has real descriptions for the resource and its fields and was regenerated, not
pasted; `jsonschema.json`, validation files, refschema output and PyDABs models follow from a
generator run (`./task generate-schema`, `generate-direct`, `pydabs-codegen`). Launch stage: the
resource-level `x-databricks-launch-stage` comes only from `launchStageOverrides` in
`bundle/internal/annotation/preview.go`, so a resource without an entry ships in PyDABs whatever
its fields' stages, like every non-Postgres resource today. Whether that is intended is a question
to the author, not an upstream ask.
Table: `generated file | consistent with source? | note`

### P8. User-facing text

Changelog fragment under `.nextchanges/bundles/` in the required one-line form; approval messages
short and actionable; error strings without trailing punctuation. The PR description is read for
intent only: the code carries decisions, API requests and test coverage, so a description that does
not restate them is not a finding.
Table: `text (file:line) | issue | suggested wording`

### P9. Comments and size

Comments on the adapter explain what the API does, not how the engine works. Multi-paragraph
comments, comments that restate code, and comments describing planner internals get a "delete" or
"move to the README" comment. Prose titles in scripts get the same.
Table: `location | what the comment explains | keep / delete / move`

### P10. Consistency with siblings

Resources copy their exemplar, so a block that repeats a sibling's (a setup function in a test
script, a UC-name resolution loop in a mutator, an approval message, a fake handler) is the expected
outcome, not a finding. For each repeated block, open the sibling and compare shape: same shape, no
comment; different shape with no comment saying why, one comment that points at the sibling and asks
for the same shape. When the repo already has a helper for the job (`acceptance/bin/`,
`acceptance/script.prepare`, `libs/`), ask for that helper by name. Do not ask for a new shared
helper, a new `script.prepare`, or a `libs/` function to remove the repetition; that is a separate
PR if anyone wants it.
Table: `block (file:line) | sibling it follows | same shape? | if not, reason given?`

Skip formatting, identifier naming, anything a linter catches, defensive-check debates beyond what
P3 lists, repetition that matches its sibling, and the completeness of the PR description.

## 4. Red flags

Any of these is a top-of-review comment and usually a redesign at the API:

| Seen in the diff | Means |
|---|---|
| `KeyedSlices` key assembled from several fields or a non-field name | Elements have no client identity; ask the API for a key field |
| `OverrideChangeDesc` in a new resource | A yml rule or an API annotation is missing |
| `DoUpdate` whose mask excludes most of the config, or whose endpoint only removes | API update semantics do not fit declarative config |
| Hand-written `recreate_on_changes` or `ignore_remote_changes` without a comment and upstream ask | Annotation missing in the proto |
| A mirror config struct rather than the embedded SDK type | Schema drifts from the API |
| Fake over ~150 lines, or validating field combinations | Backend policy modelled locally, unverified |
| More than five acceptance directories for one resource | Backend refusal matrix in the tests |
| Invariant template without a matrix entry | Dead test |
| Approval group in `deploy.go` only | Destroy loses the warning |
| Engine or `libs/structs` changes in a resource PR | Workaround for an API gap; split and discuss |

## 5. Write the comments

Every confirmed problem gets a comment; no target count. Each comment points at one line, states
the behaviour with a concrete input and outcome, names the precedent or rule it relies on, and ends
with the change or the question. One to three sentences, plain, no headings or severity labels
inside the comment. Prefer a question where the author may know something you do not. No praise,
no verdict, no summary.

Order: redesign and scope first, then correctness, then wiring and tests, then generated files and
wording. Raise each issue once and list other locations inside that comment. The review body
carries cross-cutting asks only: the split, and a design question that spans files. Asks to
document decisions, API requests or test runs in the description are not comments; the code is
leading, and the inline comment on the rule, mask or exclusion is where that fact lives.

Before printing, re-check every comment against the code, drop any that misread it, and make sure
every worksheet problem row is either a comment or listed under "Not raised" with the reason.

## 6. Output

Print, do not post:

```
## Inline comments

### path/to/file.go:LINE
<comment>

## Review body
<cross-cutting asks, or nothing>

## Evidence (not for posting)
- path/to/file.go:LINE: <what backs each comment>

Not raised:
- <worksheet row>: <why>

## Worksheet (not for posting)
<the tables from P1 to P10, plus the triage numbers>
```

Use line numbers from `TARGET`. Only post if asked, and never to a PR you were told to dry-run.
