---
name: author-new-resource
description: Add a new resource type to Declarative Automation Bundles (the resources map in databricks.yml) on the direct deployment engine, for engineers who own an API but do not know the CLI codebase. Starts with an API readiness gate that sends API-shape problems back to the proto, then walks the wiring, generators, tests and PR in fixed phases with a verification command at each step. Use when asked to add, onboard or support a new bundle resource, or when an API team asks what DABs needs from their API.
---

# Author a new bundle resource

One resource per PR, roughly 250 to 400 hand-written Go lines, an adapter of 60 to 130 lines with
no plan overrides, a fake of 70 to 100 lines, three to five acceptance directories. That is what the
last several resources looked like and what review expects. If your resource needs more, the API
is usually the reason, and the API is where to fix it.

Work in phases. Each phase ends with a check. Do not start the next phase until the check passes.

References in this directory: `references/api-readiness.md` (phase 0), `references/wiring.md`
(phases 2, 3, 5 and 6), `references/testing.md` (phase 4), `references/pitfalls.md` (read once
before phase 2, and again before phase 6). Repo rules under `.agents/rules/`,
`bundle/config/resources/README.md` (the files a new type needs) and
`bundle/direct/dresources/README.md` (the adapter) apply as well; where this skill and a README
disagree, the README is the contract and this skill is the path through it.

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

Decide and write down in your working notes (the code records each of these from phase 2 on; the PR
description does not repeat them):

1. **Names.** Bundle plural and singular (`resources.<plural>`, snake_case, matching the API's
   collection name), Go type, SDK service and types: the resource message, the create request, the
   get response. Confirm the SDK in `go.mod` has them: `grep -n "<Resource> struct" $(go env GOMODCACHE)/github.com/databricks/databricks-sdk-go@*/service/<pkg>/model.go`.
   If the SDK lacks the API, that is a separate SDK bump PR first; see `.agents/skills/bump-sdk`.
   Embed the message GET returns in the config type, so state and remote share one type and
   `RemapState` is not needed; server-only fields on it are covered by the generated output-only
   rules. Two exceptions: the create request carries inputs the resource message lacks, or the
   resource message marks a server-assigned field `REQUIRED` (it would become a required config
   key). In both cases embed the create request, pin the resource type with `api_type` in
   `configs/<plural>.yml` so the generated rules still describe the read shape, and for the second
   case ASK for `OUTPUT_ONLY` upstream.
2. **Identity.** Which field or fields the engine stores as the id, how `DoRead` gets from id to a
   Get request, and whether changing an identity field recreates (`provided_id_fields`) or renames
   (`updatable_id_fields`).
3. **Mutability.** Which fields update in place, which are immutable (should already be annotated
   after phase 0), whether the update mask is `*` or a documented list, whether there is an update
   at all.
4. **Asynchrony.** Does create, update or delete return an operation; is the id available before
   completion; is the resource readable right after create.
5. **URL.** Does the workspace UI have a page for one instance; what is its path.
6. **Launch stage.** Field stages come from the proto. The resource's own stage is a CLI decision,
   declared by `SchemaLaunchStage()` on the config type, and decides PyDABs eligibility: no
   declaration means a public Python package (readiness check 2.10).
7. **Exemplar.** Pick the existing resource closest to your shape and copy its files, do not write
   from scratch. Current good choices on `main`:
   - server-assigned id, plain CRUD: `instance_pools.go` in `bundle/direct/dresources/` and
     `bundle/config/resources/`, fake `libs/testserver/instance_pools.go`;
   - full-name identity derived from parent fields: `features.go` (short `configs/features.yml`);
   - parent plus leaf AIP name: `mcp_services.go` with `configs/mcp_services.yml`;
   - composed id from two user fields, no UI page: `data_quality_monitors.go` with
     `configs/data_quality_monitors.yml`;
   - asynchronous create with the id available at once: `database_instances.go` (returns the id
     from the create response and polls in `WaitAfterCreate`).
   `git ls-files '*<exemplar plural>*'` lists the exemplar's files, its per-type test files
   included.
   If one of these does not exist at your base commit, take the newest resource that passes the
   phase 2 rules: no `OverrideChangeDesc`, no inline wait loop, a static or `*` mask, a fake under
   150 lines. Never copy an adapter because it is recent; copy it because it is clean.
   Copy the exemplar's shape block for block, test scripts and fake included, and keep that shape
   even where two resources now carry the same block: review checks that your copy matches the
   sibling, not that it is unique. Use a helper when the repo already has one for the job
   (`acceptance/bin/`, `acceptance/script.prepare`, `libs/`); do not add one to remove the
   repetition, that is a separate PR.

Check: the seven items are written down and the exemplar's files are open side by side.

## Phase 2: wire the resource

Follow `references/wiring.md` sections 1 and 2 in order: the field in `bundle/config/resources.go`,
the config type with its declarations, the bind fixture, the target mode examples, the adapter and
its test fixture, `configs/<plural>.yml` with only the rules the proto cannot express (each with a
one-line comment naming the API behaviour), the fake server. The adapter, fake and per-type test
files register themselves from `init()`; no shared list needs the type.

Rules that review enforces on the adapter:

- One SDK call per `DoCreate`, `DoUpdate`, `DoDelete`. Return SDK errors unchanged; the engine adds
  the resource key and id.
- Helper methods are unexported. `NewAdapter` in `adapter.go` rejects wrong method signatures and
  any exported method that `IResource` does not declare.
- Asynchronous waits go in `WaitAfterCreate`, `WaitAfterUpdate`, `WaitAfterDelete`, so state is
  persisted before polling.
- `RemapState` is a field copy; any remapping lives in `DoRead`.
- No nil checks on arguments the framework guarantees. `DoCreate` must return a non-empty id; if
  the API can complete without one, return an error.
- No `OverrideChangeDesc`. If you believe you need it, go back to phase 0.
- Update mask `*` when the API accepts it, otherwise a static list with a comment.

Check:

```bash
go build ./... && go test ./bundle/config ./bundle/config/resources ./bundle/config/mutator/resourcemutator ./bundle/phases -count=1
```

These packages hold the tests that require the config type's declarations, the bind fixture and
the target mode examples, so they fail until each exists. The other decisions of wiring section 3
are checked later or not at all: create recovery in phase 3, deployment history by the `DMS=true`
acceptance leg in phase 4, launch stage and etag protection by hand (wiring section 6). Make each
decision now; phase 5 reviews them.

The adapter tests need the generated rules from phase 3 (input-only fields fail `TestAll` until
then), so they are part of the phase 3 check.

## Phase 3: generate

Run the generators in the order given in `references/wiring.md` section 4, document the
placeholders in `bundle/internal/schema/annotations/<plural>.generated.yml` in
`annotations/<plural>.yml`, run `./task generate-schema` again, then `./task generate-check`. Read
`configs/<plural>.generated.yml` back against the API docs (section 5 of the wiring reference);
anything you had to hand-write in `configs/<plural>.yml` that should have been generated is an
upstream annotation to request, if phase 0 did not already.

Check:

```bash
go test ./bundle/direct/dresources -run 'TestAll/^<plural>$|TestEveryResourceTypeHasAdapter|TestEveryAdapterHasTestFixture|TestCreateRecoveryCoverage|TestResourceConfig|TestGeneratedResourceConfig|TestConfigFiles|TestResourcesYML|TestInputSubset|TestRemoteSuperset|TestNameStableOrOptedOut|TestNoUpdateResourcesCoverAllFields' -v -count=1
go test ./bundle/internal/schema -run TestRequiredAnnotationsForNewFields -count=1
```

plus: `./task generate-check` leaves no diff, `bundle/internal/schema/annotations/<plural>.generated.yml`
does not exist, `acceptance/bundle/refschema/output/<plural>.txt` exists (the first
`generate-refschema` run reports the new file as unexpected; run it again), and
`python/databricks/bundles/<plural>/` exists exactly when you decided it should in phase 1 item 6:
without a `SchemaLaunchStage()` declaration the package is generated; with `PRIVATE_PREVIEW` or
`PUBLIC_BETA` a regeneration removes it.

## Phase 4: tests

Follow `references/testing.md`. Add the canonical acceptance set (`basic`, `lifecycle` with
`Cloud = true`, `remote-delete`, `deployment/bind/<plural>`, the invariant config and the
exclusions for the legs it cannot run, the PyDABs support directory when eligible). The coverage
tests in testing reference section 1 name every per-type file that is still missing. Copy the
exemplar's directories and adapt them; keep `title` lines to one line and let `print_requests.py`,
`replace_ids.py` and `musterr` do the work.

Then prove the tests ran, with the commands in testing reference section 4. `Cloud = true` on the
`lifecycle` directory is what gets it run against a real workspace: the PR's Integration Tests check
selects every changed cloud test once the PR is open. Testing reference section 6 says how to run
it locally first when you have access and want a faster loop.

Check: every proof command lists your tests; `./task test` passes locally; once the PR is open, its
Integration Tests check passes.

## Phase 5: bundle-wide decisions

Go through `references/wiring.md` section 3 and confirm each decision explicitly: run_as, presets,
permissions, grants, UC dependencies, approval prompts in both deploy and destroy, workspace URL,
launch stage, bind, create recovery, deployment-history allowlist or exclusion, etag protection.
Each decision is recorded where the code records it: the declaration in
`bundle/config/resources/<plural>.go`, the registration in the per-type test file, the `-exclude.toml`
or `EnvMatrixExclude` line, with a one-line comment when the reason is not obvious from its
neighbours. "Not applicable" is a valid decision when that comment says why.

Check: the phase 2 and phase 3 test commands pass, and every row of wiring section 6 except the
last two (the changelog fragment and the PR scope, which phase 6 checks) is verified.

## Phase 6: the PR

- Scope: this resource and nothing else. No changes to other resources, no new config syntax, no
  engine changes, and no changes to shared `libs/` beyond the fake server's file in
  `libs/testserver` (and `dmsResourceTypes`, when the deployment-history service accepts the type).
  If you found something worth fixing elsewhere, open a separate PR.
- Changelog fragment `.nextchanges/bundles/<plural>.md`, one line ending with the PR link; fill in
  the PR number right after opening the PR.
- Run `.agents/skills/pr-checklist`.
- Description, three sections from the PR template, short; the diff carries the detail:
  - Changes: what the resource is, and its identity and update shape, in two sentences.
  - Why: one sentence.
  - Tests: the acceptance directories, one line. Cloud coverage is visible from `Cloud = true` and
    the Integration Tests check; decisions and API requests live in code comments. Do not restate
    them here.
- Read `references/pitfalls.md` once more against your diff.

Check: the last two rows of wiring section 6 are verified: the changelog fragment exists, and the
diff touches only the files in the wiring reference plus your acceptance directories. The
hand-written Go line count is in the expected range; if it is not, say why in the description.

## When the API changes under you

If the proto changes while the PR is open (an annotation you asked for lands, or a field moves),
bump the SDK in a separate PR, rerun phase 3, and delete the hand-written rule the annotation now
covers. `TestResourcesYMLNoRedundantRules` fails until you do.
