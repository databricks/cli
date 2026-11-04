---
name: author-new-resource
description: Add a new resource type to Declarative Automation Bundles (resources.<plural> in databricks.yml) on the direct deployment engine, for engineers who own an API but do not know the CLI codebase. Starts with an API readiness gate that sends API-shape problems back to the proto, then walks the wiring, generators, tests and PR in fixed phases with a verification command at each step. Use when asked to add, onboard or support a new bundle resource, or when an API team asks what DABs needs from their API.
---

# Author a new bundle resource

One resource per PR, roughly 250 to 400 hand-written Go lines, an adapter of 60 to 130 lines with
no plan overrides, a fake of 70 to 100 lines, three to five acceptance directories. That is what the
last several resources looked like and what review expects. If your resource needs more, the API
is usually the reason, and the API is where to fix it.

Work in phases. Each phase ends with a check. Do not start the next phase until the check passes.

References in this directory: `references/api-readiness.md` (phase 0), `references/wiring.md`
(phases 2 and 5), `references/testing.md` (phase 4), `references/pitfalls.md` (read once before
phase 2, and again before phase 6). Repo rules under `.agents/rules/` and
`bundle/direct/dresources/README.md` apply as well; where this skill and the README disagree, the
README is the contract and this skill is the path through it.

## Phase 0: API readiness gate

Before writing code, run every check in `references/api-readiness.md` against your proto (or the
pinned `.codegen/cli.json` if you have no universe checkout). Write the verdict table into your
notes: one line per check, STOP, ASK or NOTE, with the field or RPC concerned.

- Any STOP: send the note from the reference to your API owners and stop here. Do not build a
  composite key, a plan override, or an engine change to compensate. Those will be rejected in
  review and cost more than the proto change.
- Any ASK: file the proto change now, before the PR, and carry the smallest commented bridge the
  reference allows.
- All NOTE: continue.

Check: the verdict table exists and has no STOP.

## Phase 1: establish the contract

Decide and write down, in the PR description draft:

1. **Names.** Bundle plural and singular (`resources.<plural>`, snake_case, matching the API's
   collection name), Go type, SDK service and types: the resource message, the create request, the
   get response. Confirm the SDK in `go.mod` has them: `grep -n "<Resource> struct" $(go env GOMODCACHE)/github.com/databricks/databricks-sdk-go@*/service/<pkg>/model.go`.
   If the SDK lacks the API, that is a separate SDK bump PR first; see `.agents/skills/bump-sdk`.
   Embed the message GET returns in the config type, so state and remote share one type and
   `RemapState` is not needed; server-only fields on it are covered by the generated output-only
   rules. Two exceptions: the create request carries inputs the resource message lacks, or the
   resource message marks a server-assigned field `REQUIRED` (it would become a required config
   key). In both cases embed the create request, pin the resource type in `apitypes.yml` so the
   generated rules still describe the read shape, and for the second case ASK for `OUTPUT_ONLY`
   upstream.
2. **Identity.** Which field or fields the engine stores as the id, how `DoRead` gets from id to a
   Get request, and whether changing an identity field recreates (`provided_id_fields`) or renames
   (`updatable_id_fields`).
3. **Mutability.** Which fields update in place, which are immutable (should already be annotated
   after phase 0), whether the update mask is `*` or a documented list, whether there is an update
   at all.
4. **Asynchrony.** Does create, update or delete return an operation; is the id available before
   completion; is the resource readable right after create.
5. **URL.** Does the workspace UI have a page for one instance; what is its path.
6. **Launch stage** and therefore PyDABs eligibility (GA and public preview only).
7. **Exemplar.** Pick the existing resource closest to your shape and copy its files, do not write
   from scratch. Current good choices on `main`:
   - server-assigned id, plain CRUD: `instance_pool.go` in `bundle/direct/dresources/` and
     `bundle/config/resources/`, fake `libs/testserver/instance_pools.go`;
   - full-name identity derived from parent fields: `feature.go` (three-line rules file);
   - parent plus leaf AIP name: `mcp_service.go` with `configs/mcp_services.yml`;
   - composed id from two user fields, no UI page: `data_quality_monitor.go` with
     `configs/data_quality_monitors.yml`;
   - asynchronous create with the id available at once: `database_instance.go` (returns the id
     from the create response and polls in `WaitAfterCreate`).
   If one of these does not exist at your base commit, take the newest resource that passes the
   phase 2 rules: no `OverrideChangeDesc`, no inline wait loop, a static or `*` mask, a fake under
   150 lines. Never copy an adapter because it is recent; copy it because it is clean.

Check: the seven items are written down and the exemplar's files are open side by side.

## Phase 2: wire the resource

Follow `references/wiring.md` sections 1 and 2 in order: config type, registrations, adapter,
`all.go`, `all_test.go` fixture, `configs/<plural>.yml` with only the rules the proto cannot express
(each with a one-line comment naming the API behaviour), fake server from section 4.

Rules that review enforces on the adapter:

- One SDK call per `DoCreate`, `DoUpdate`, `DoDelete`. Return SDK errors unchanged; the engine adds
  the resource key and id.
- Asynchronous waits go in `WaitAfterCreate`, `WaitAfterUpdate`, `WaitAfterDelete`, so state is
  persisted before polling.
- `RemapState` is a field copy; any remapping lives in `DoRead`.
- No nil checks on arguments the framework guarantees. `DoCreate` must return a non-empty id; if
  the API can complete without one, return an error.
- No `OverrideChangeDesc`. If you believe you need it, go back to phase 0.
- Update mask `*` when the API accepts it, otherwise a static list with a comment.

Check:

```bash
go build ./... && go test ./bundle/config -count=1
```

The adapter tests need the generated rules from phase 3 (input-only fields fail `TestAll` until
then), so they are part of the phase 3 check.

## Phase 3: generate

Run the generators in the order given in `references/wiring.md` section 3, fill the
`annotations.yml` placeholders for your type, run `./task generate-schema` again, then
`./task generate-check`. Read `configs/<plural>.generated.yml` back against the API docs (section 7
of the wiring reference); anything you had to hand-write in `configs/<plural>.yml` that should have
been generated is an upstream annotation to request, if phase 0 did not already.

Check:

```bash
go test ./bundle/direct/dresources -run 'TestAll/^<plural>$|TestResourceConfig|TestConfigFiles|TestResourcesYML|TestInputSubset|TestRemoteSuperset|TestNameStableOrOptedOut|TestNoUpdateResourcesCoverAllFields' -v -count=1
```

plus: `./task generate-check` is clean, `grep -c PLACEHOLDER bundle/internal/schema/annotations.yml`
did not grow, `acceptance/bundle/refschema/output/<plural>.txt` exists (the first
`generate-refschema` run reports the new file as unexpected; run it again), and
`python/databricks/bundles/<plural>/` exists exactly when the resource's launch stage makes it
eligible (GA or public preview). A preview resource that produced a Python package needs the
launch-stage override from the readiness reference, then a regeneration that removes the package.

## Phase 4: tests

Follow `references/testing.md`. Add the canonical acceptance set (`basic`, `lifecycle` with
`Cloud = true`, `remote-delete`, `deployment/bind/<singular>`, the invariant template and its matrix
entry, the PyDABs support directory when eligible) and extend the completeness tests the wiring
reference lists. Copy the exemplar's directories and adapt them; keep `title` lines to one line and
let `print_requests.py`, `replace_ids.py` and `musterr` do the work.

Then prove the tests ran, with the commands in testing reference section 4, and run the `lifecycle`
test against a real workspace (section 6). Record which environment.

Check: every proof command lists your tests; `./task test` passes locally; the cloud run passed.

## Phase 5: bundle-wide decisions

Go through `references/wiring.md` section 5 and make each decision explicitly: run_as, presets,
permissions, UC dependencies, approval prompts in both deploy and destroy, workspace URL, state
fixture, deployment-history allowlist or exclusion. Write each decision and its reason into the PR
description. "Not applicable" is a valid decision when you say why.

Check: the completeness tests named in that section pass; the PR description lists the decisions.

## Phase 6: the PR

- Scope: this resource and nothing else. No changes to other resources, no new config syntax, no
  engine changes, and no changes to shared `libs/` beyond the fake server in `libs/testserver` and
  the URL pattern in `libs/workspaceurls`. If you found something worth fixing elsewhere, open a
  separate PR.
- Changelog fragment `.nextchanges/bundles/<plural>.md`, one line.
- Run `.agents/skills/pr-checklist`.
- Description, three sections from the PR template:
  - Changes: the resource, its identity and update shape in two sentences, the decisions from
    phase 5, the API changes requested in phase 0 and their status.
  - Why: one sentence.
  - Tests: the acceptance directories, which ran on cloud and where, the proof that the invariant
    config ran.
- Read `references/pitfalls.md` once more against your diff.

Check: the diff touches only the files in the wiring reference plus your acceptance directories, and
the hand-written Go line count is in the expected range. If it is not, say why in the description.

## When the API changes under you

If the proto changes while the PR is open (an annotation you asked for lands, or a field moves),
bump the SDK in a separate PR, rerun phase 3, and delete the hand-written rule the annotation now
covers. `TestResourcesYMLNoRedundantRules` fails until you do.
