# Testing a new resource

Every item here ends with a command that proves the test exists and ran. A test that is written but
never selected is the most common gap in resource PRs, so run the proof commands, not just the
tests.

## 1. Go tests that must know about the type

Most of these are coverage tests that run over every resource type and name the one whose per-type
file or declaration is missing. That is intended: the failure tells you which file you have not
added or which decision you have not made yet. None of them needs an entry in a shared list.

| Test | What it enforces | Run alone |
|---|---|---|
| `TestAll` in `bundle/direct/dresources/all_test.go` | Create, read, update, delete through the adapter against the fake, round-trip of every field you set in the fixture that `<plural>_test.go` registers with `registerTestConfig` (or `registerTestDeps` if the fake needs a parent object first). Populate every field; a sparse fixture proves little. | `go test ./bundle/direct/dresources -run 'TestAll/^<plural>$' -v -count=1` |
| `TestEveryResourceTypeHasAdapter`, `TestEveryAdapterHasTestFixture`, `TestCreateRecoveryCoverage` in the same package | The adapter is registered with `registerAdapter`; the type and its `.permissions` or `.grants` register a test fixture; a failed create request is recovered from or opted out with a reason. | `go test ./bundle/direct/dresources -run 'TestEveryResourceTypeHasAdapter\|TestEveryAdapterHasTestFixture\|TestCreateRecoveryCoverage/^<plural>$' -v -count=1` |
| `TestNoUpdateResourcesCoverAllFields` in `all_test.go` | For a type without `DoUpdate`: every settable field is covered by a recreate, id or ignore rule, so no change can plan an unsupported update. | `go test ./bundle/direct/dresources -run 'TestNoUpdateResourcesCoverAllFields/^<plural>$' -v -count=1` |
| `TestNameStableOrOptedOut` (same file) | A backend-assigned `name` on a type with `DoUpdate` is declared in `stable_output_fields`. | `go test ./bundle/direct/dresources -run TestNameStableOrOptedOut -v -count=1` |
| `TestInputSubset`, `TestRemoteSuperset` in `type_test.go` | Config fields exist in the state type; state fields exist in the remote type. Exceptions are registered from `<plural>_test.go` with `registerKnownMissingInStateType` or `registerKnownMissingInRemoteType` and must have a comment. | `go test ./bundle/direct/dresources -run 'TestInputSubset\|TestRemoteSuperset' -v -count=1` |
| `TestResourceConfig`, `TestGeneratedResourceConfig`, `TestConfigFiles`, `TestResourcesYML*` in `all_test.go` and `config_test.go` | Every rule in `configs/<plural>.yml` resolves to a real field, no rule duplicates the generated file, the file name is a registered type. | `go test ./bundle/direct/dresources -run 'TestResourceConfig\|TestConfigFiles\|TestResourcesYML' -v -count=1` |
| `TestCustomMarshallerIsImplemented`, `TestResourcesAllResources` in `bundle/config/resources_test.go` | The config type has value-receiver `MarshalJSON` and pointer-receiver `UnmarshalJSON`; its `ResourceDescription()` plural equals its key under `resources:`. | `go test ./bundle/config -run 'TestCustomMarshallerIsImplemented\|TestResourcesAllResources' -v -count=1` |
| The declaration tests: `TestRunAsSupportIsDeclaredForEveryResourceType`, `TestEveryResourceTypeDeclaresApproval`, `TestApprovalDeclarations`, `TestEveryResourceTypeDeclaresWorkspaceURL`, `TestResourceTypesWithWorkspacePageInitializeURL`, `TestPermissionsDeclaredIffPermissionsField`, `TestGrantsDeclaredIffGrantsField`, `TestResourceTypesWithUCNameFieldsDeclareThem`, `TestLifecycleStartedTelemetryIsDeclaredByTypesWithStarted` | The config type declares each fact of wiring section 3 that applies to it. | `go test ./bundle/config ./bundle/config/mutator/resourcemutator ./bundle/phases -run 'Declare\|Declaration\|WorkspacePageInitializeURL' -v -count=1` |
| `TestResourcesBindSupport` in `bundle/config/resources/bind_test.go` | `<plural>_bind_test.go` registers a bind fixture, whose `Exists` works through a strict mock with an expectation for the GET call it makes, or a reason why the type cannot be bound. | `go test ./bundle/config/resources -run TestResourcesBindSupport -v -count=1` |
| `TestAllResourcesMocked`, `TestAppropriateResourcesAreRenamed`, `TestProcessTargetMode*` in `bundle/config/mutator/resourcemutator/apply_target_mode_test.go` | `<plural>_target_mode_test.go` registers examples of the type with `registerTargetModeExamples`; a user-facing name gets the dev prefix unless the type implements `NamePrefixExemptReason()`. | `go test ./bundle/config/mutator/resourcemutator -run 'TestAllResourcesMocked\|TestAppropriateResourcesAreRenamed\|TestProcessTargetMode' -v -count=1` |
| `TestStateToBundle*` in `bundle/statemgmt/state_load_test.go` | State with the type loads back into config with id and modified status. The tests build a resource of every type themselves; there is nothing to add. | `go test ./bundle/statemgmt -run TestStateToBundle -v -count=1` |
| `TestRequiredAnnotationsForNewFields` in `bundle/internal/schema` | Every field of the type has a description, from upstream or from `annotations/<plural>.yml`. | `go test ./bundle/internal/schema -run TestRequiredAnnotationsForNewFields -v -count=1` |
| `TestResourceTestDirsCoverage` in `acceptance/resource_test_dirs_test.go` | `acceptance/bundle/resources/<plural>/` and `acceptance/bundle/deployment/bind/<plural>/` exist, or the type is in `LackingResourceTests` or `LackingBindTests` with the reason. It does not check what the directories hold. | `go test ./acceptance -run '^TestResourceTestDirsCoverage$' -v -count=1` |
| `TestInvariantConfigsCoverage`, `TestInvariantConfigNames` in `acceptance/invariant_test.go` | `invariant/configs/<plural>.yml.tmpl` defines a resource of the type, and some config attaches permissions or grants when the type has them. | `go test ./acceptance -run '^TestInvariantConfig' -v -count=1` |
| `test_python_support_coverage`, `test_has_all_test_cases`, `test_core_public_api` under `python/databricks_tests/core/` | PyDABs has a support fixture, a generated test case and an updated public API snapshot for the type (only when PyDABs generates it; see readiness check 2.10). | `cd python && uv run pytest databricks_tests/core -q` |

Use `-count=1` so Go does not report a cached result as a run.

## 2. The acceptance set

Directory layout under `acceptance/bundle/resources/<plural>/` plus three directories elsewhere.
Copy the most recent resource's directories and adapt; do not invent new shapes.

| Directory | Covers | Settings |
|---|---|---|
| `resources/<plural>/basic/` | `bundle summary` before and after deploy with id and URL, the create request body via `print_requests.py`, a redeploy that plans nothing, an in-place update, an immutable change that plans a recreate and is then deployed with `--auto-approve`, destroy with the delete request. | Local only. `RecordRequests` is inherited from the parent `resources/test.toml`. |
| `resources/<plural>/lifecycle/` | Create, update, plan clean, destroy against a real workspace. Uses `$UNIQUE_NAME` in every name, creates its own prerequisites, cleans up in a trap. | `Cloud = true`, `RecordRequests = false`. This is the live run the resource README requires. |
| `resources/<plural>/remote-delete/` | Delete the object out of band with the CLI's own command, then `bundle plan` must detect it is gone and plan a create. | Local only. |
| `deployment/bind/<plural>/` | Create the object outside the bundle with the CLI's own create command against the fake, `bundle deployment bind` it, summary, `unbind`, summary. | Local only. Older bind tests stub the object's GET with a `[[Server]]` block instead; do not copy that, the fake already models the route. |
| `invariant/configs/<plural>.yml.tmpl` | Every invariant test (no-drift, delete-idempotent, destroy-idempotent, the migrations) runs every config in that directory. Add `-init.sh` and `-cleanup.sh` next to it if the resource needs a parent object the harness does not provide. | Exclude the legs the config cannot run in `<plural>.yml.tmpl-exclude.toml`, one `[[Exclude]]` per reason (see `acceptance/bundle/invariant/README.md`): `When = ["DMS=true"]` if the type is not in the deployment-history allowlist, `When = ["CONFIG_Cloud=true"]` if the config cannot run on cloud, `Tests = ["migrate", "auto-migrate"]` for a direct-only type (they start from a terraform deployment), `Tests = ["continue_293"]` if that leg, which deploys first with an older CLI, fails. |
| `python/<plural>-support/` | The PyDABs model loads from YAML and from Python and a mutator runs. Six files; copy the newest sibling directory and follow `acceptance/bundle/python/README.md`. | `EnvMatrix.PYDAB_VERSION = ["current"]` for a brand-new type. Only for types PyDABs generates (no `SchemaLaunchStage()` below public preview). |
| `refschema/output/<plural>.txt` | Generated listing of the type's fields and which are state. | Regenerated by `-update`; commit it. |

Script conventions, all enforced by review: wrap every command in `trace`; one-line `title` per
phase; `musterr` for a command that must fail; `print_requests.py //<api-path>` instead of inline
`jq` on the request log; `replace_ids.py` right after the first deploy; project any API read you
print to the fields you assert; commit static inputs instead of generating them in the script.

Choose updates that exercise the resource's main payload. Changing only a description does not
verify a nested definition's replacement, clearing, or drift repair. Where relevant, extend the
lifecycle with a content edit, removal of an optional nested value, and a clean subsequent plan;
use at least two elements when testing list ordering. Distinguish a server that reorders accepted
input from one that rejects noncanonical input, using API docs or cloud evidence before asserting
permanent drift. Test etag conflicts before planning, the warning/refusal/explicit-overwrite flow,
and saved-plan rejection as described in readiness check 2.11. Development presets need an
acceptance case showing the resource's displayed name, not only a fixture entry.

Every `EnvMatrix` variant must produce the same `output.txt`. The bundle tree runs a `DMS=` and a
`DMS=true` leg by default; a type absent from `dmsResourceTypes` in
`libs/testserver/bundledeployments.go` fails the second leg on the first deploy, so either add it
there (the list mirrors the service's allowlist; coordinate the backend change) or exclude the leg
with a comment that says why.

## 3. The fake server

Scope rule: model what the CLI's plan and apply depend on, and nothing else.

- Create returns the object with server-assigned fields filled (id, timestamps, derived names).
- Get returns the stored object; fields the real API never echoes are stripped only when the proto
  marks them input-only.
- Update applies the mask the adapter sends (`*` means replace the writable fields and keep the
  server-assigned ones).
- Delete removes the object. A missing object returns 404 with a JSON body carrying `error_code`
  (`RESOURCE_DOES_NOT_EXIST`), a duplicate create 409 (`RESOURCE_ALREADY_EXISTS`), bad input 400
  (`INVALID_PARAMETER_VALUE`). The SDK maps these to typed errors only when the body is JSON.

Mechanics: one file `libs/testserver/<api>.go` per API, whose `init()` calls `registerStore` (a
store with a map keyed by id, which handlers reach through `stateOf`) and `registerRoutes`, both
keyed by the file name; every handler including delete starts with `defer s.LockUnlock()()`.
For a type with permissions, the same `init()` calls `registerPermissionsObjectType` with the
request and response object types that `PermissionsSupport()` declares
(`TestPermissionsObjectTypesMatchTestServer`).
For an asynchronous API, the fake returns a pending operation at
least once before completing, so the adapter's polling runs locally. `MapGet` and `MapDelete` cover get and delete when the stored body is the
request body. Expect 70 to 100 lines. A fake several times that size is modelling backend policy
the CLI does not branch on; move that coverage to the cloud test and cite the server code in the
PR instead.

Do not encode failure triggers in fixture data. Transport and response faults come from
`acceptance/bin/fault.py` and `kill_after.py`, which the harness already provides.

## 4. Proving a test ran

```bash
# every subtest for the resource, with its matrix variants
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -v -count=1 2>&1 | grep -E -- '--- (PASS|SKIP|FAIL): '

# the invariant suite actually selected your config
go test ./acceptance -run 'TestAccept/bundle/invariant/no_drift' -v -count=1 2>&1 | grep 'INPUT_CONFIG=<plural>.yml.tmpl'

# the DMS leg runs (or is deliberately excluded); counts passing subtests, not log lines
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -v -count=1 2>&1 | grep -c -- '--- PASS.*DMS=true'

# effective settings after inheritance
cat acceptance/bundle/resources/<plural>/*/out.test.toml
```

A test that does not appear in the first listing was not selected. Tests under `Cloud = true` also
run locally; the cloud run is additional. On the PR, the Integration Tests check runs every changed
`Cloud = true` test; its job log contains a line `Selected N changed tests (...)` followed by `PASS`
and `SKIP` lines with reasons. Confirm your `lifecycle` test is in the `PASS` lines. Ask a
maintainer for the log if you cannot open the run.

## 5. Regenerating goldens

```bash
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -update -count=1
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -count=1
```

Run without `-update` afterwards and require a clean pass. Never run `-update` while two matrix
variants produce different output; fix the variant first. `./task test-update` regenerates the
whole suite and can pick up unrelated drift; revert any output file you did not mean to change.
Regenerate `out.test.toml` files alone with
`go test ./acceptance -run '^TestAccept$' -only-out-test-toml`.

## 6. The cloud run

A new type needs at least one test executed against a real workspace before the PR merges.
`Cloud = true` on the `lifecycle` directory is what makes that happen: the PR's Integration Tests
check selects the changed cloud tests and runs them, and the check is the record. Nothing about the
run goes in the description.

To run it before the PR opens, when you have access:

```bash
deco env run -i -n aws-prod-ucws -- go test ./acceptance -run 'TestAccept/bundle/resources/<plural>/lifecycle' -v -count=1
```

If the test cannot run on one cloud, narrow it with `CloudEnvs` in its `test.toml` and say why in a
comment there.

## 7. Changelog fragment

One new file `.nextchanges/bundles/<plural>.md`, one line, imperative, ending with the PR link:

```
* Add the `<plural>` resource for <one phrase on what it manages>. ([#NNNN](https://github.com/databricks/cli/pull/NNNN))
```

Never edit `CHANGELOG.md` or `.nextchanges/version`.

## 8. Before opening the PR

Follow `.agents/skills/pr-checklist/SKILL.md`: `./task fmt`, `./task checks`, `./task lint`,
`./task test`, the generators you touched (`./task generate-schema`, `./task generate-direct`,
`./task pydabs-codegen pydabs-test pydabs-lint pydabs-docs` when Python changed), then the cleanup
scan for debug prints and stray files. Use the PR template's three sections and keep the
description short; the diff carries the detail.
