# Testing a new resource

Every item here ends with a command that proves the test exists and ran. A test that is written but
never selected is the most common gap in resource PRs, so run the proof commands, not just the
tests.

## 1. Go tests that must know about the type

Most of these are completeness tests that fail until you add the type. That is intended: the
failure tells you which decision you have not made yet.

| Test | What it enforces | Run alone |
|---|---|---|
| `TestAll` in `bundle/direct/dresources/all_test.go` | Create, read, update, delete through the adapter against the fake, round-trip of every field you set in `testConfig`. Add a `testConfig` entry with every field populated; a type without one runs with a zero config and proves nothing. Add `testDeps` if the fake needs a parent object first. | `go test ./bundle/direct/dresources -run 'TestAll/^<plural>$' -v -count=1` |
| `TestNoUpdateResourcesCoverAllFields` (same file) | For a type without `DoUpdate`: every settable field is covered by a recreate, id or ignore rule, so no change can plan an unsupported update. | `go test ./bundle/direct/dresources -run 'TestNoUpdateResourcesCoverAllFields/^<plural>$' -v -count=1` |
| `TestNameStableOrOptedOut` (same file) | A backend-assigned `name` on a type with `DoUpdate` is declared in `stable_output_fields`. | `go test ./bundle/direct/dresources -run TestNameStableOrOptedOut -v -count=1` |
| `TestInputSubset`, `TestRemoteSuperset` in `type_test.go` | Config fields exist in the state type; state fields exist in the remote type. Exceptions are explicit lists in that file and must have a comment. | `go test ./bundle/direct/dresources -run 'TestInputSubset\|TestRemoteSuperset' -v -count=1` |
| `TestResourceConfig`, `TestGeneratedResourceConfig`, `TestConfigFiles`, `TestResourcesYML*` in `all_test.go` and `config_test.go` | Every rule in `configs/<plural>.yml` resolves to a real field, no rule duplicates the generated file, the file name is a registered type. | `go test ./bundle/direct/dresources -run 'TestResourceConfig\|TestConfigFiles\|TestResourcesYML' -v -count=1` |
| `TestCustomMarshallerIsImplemented`, `TestResourcesAllResourcesCompleteness`, `TestSupportedResources` in `bundle/config/resources_test.go` | The config type has value-receiver `MarshalJSON` and pointer-receiver `UnmarshalJSON`; it is in `AllResources()` and `SupportedResources()`. | `go test ./bundle/config -run 'TestCustomMarshallerIsImplemented\|TestResourcesAllResourcesCompleteness\|TestSupportedResources' -v -count=1` |
| `TestBundleResourcePluralNamesResolveInWorkspaceURLs` (same file) | The type has a URL pattern in `libs/workspaceurls`, or is listed in the test's no-URL map with a comment saying why there is no page. | `go test ./bundle/config -run TestBundleResourcePluralNamesResolveInWorkspaceURLs -v -count=1` |
| `TestResourcesBindSupport` (same file) | `Exists` works for the type through a strict mock; add an instance to the fixture and a mock expectation for the GET call `Exists` makes. | `go test ./bundle/config -run TestResourcesBindSupport -v -count=1` |
| `TestAllResourcesMocked`, `TestAppropriateResourcesAreRenamed`, `TestProcessTargetMode*` in `bundle/config/mutator/resourcemutator/apply_target_mode_test.go` | The dev-mode fixture has an instance of the type; a user-facing `Name` gets the dev prefix unless the type is listed as not user-named with a reason. | `go test ./bundle/config/mutator/resourcemutator -run 'TestAllResourcesMocked\|TestAppropriateResourcesAreRenamed\|TestProcessTargetMode' -v -count=1` |
| `TestRunAs*` in `run_as_test.go` | The type appears in the sorted list of all resource types, and is either in the run_as allow list or produces the "not supported" diagnostic. | `go test ./bundle/config/mutator/resourcemutator -run TestRunAs -v -count=1` |
| `TestAllResourcesExplicitlyDefinedForPermissionsSupport` in `apply_bundle_permissions_test.go` | The type is in the permissions level map or in the unsupported list. | `go test ./bundle/config/mutator/resourcemutator -run TestAllResourcesExplicitlyDefinedForPermissionsSupport -v -count=1` |
| `TestStateToBundle*` in `bundle/statemgmt/state_load_test.go` | State with the type loads back into config with id and modified status. Nothing enumerates types here, so add the fixture yourself. | `go test ./bundle/statemgmt -run TestStateToBundle -v -count=1` |
| `cmd/experimental/workspace_open_test.go` | If you added a URL pattern: the completion list, help text and unknown-type error list it. | `go test ./cmd/experimental -run 'TestBuildWorkspaceURL\|TestWorkspaceOpenCommand' -v -count=1` |
| `TestInvariantConfigsCoverage` in `acceptance/invariant_test.go` | An invariant config file exists for the type. It does not check the matrix; see section 2. | `go test ./acceptance -run '^TestInvariantConfigsCoverage$' -v -count=1` |
| `test_python_support_coverage`, `test_has_all_test_cases`, `test_public_api` under `python/databricks_tests/core/` | PyDABs has a support fixture, a generated test case and an updated public API snapshot for the type (only when the launch stage makes it eligible). | `cd python && uv run pytest databricks_tests/core -q` |

Use `-count=1` so Go does not report a cached result as a run.

## 2. The acceptance set

Directory layout under `acceptance/bundle/resources/<plural>/` plus three directories elsewhere.
Copy the most recent resource's directories and adapt; do not invent new shapes.

| Directory | Covers | Settings |
|---|---|---|
| `resources/<plural>/basic/` | `bundle summary` before and after deploy with id and URL, the create request body via `print_requests.py`, a redeploy that plans nothing, an in-place update, an immutable change that plans a recreate and is then deployed with `--auto-approve`, destroy with the delete request. | Local only. `RecordRequests` is inherited from the parent `resources/test.toml`. |
| `resources/<plural>/lifecycle/` | Create, update, plan clean, destroy against a real workspace. Uses `$UNIQUE_NAME` in every name, creates its own prerequisites, cleans up in a trap. | `Cloud = true`, `RecordRequests = false`. This is the live run the resource README requires. |
| `resources/<plural>/remote-delete/` | Delete the object out of band with the CLI's own command, then `bundle plan` must detect it is gone and plan a create. | Local only. |
| `deployment/bind/<singular>/` | Create the object outside the bundle with the CLI's own create command against the fake, `bundle deployment bind` it, summary, `unbind`, summary. | Local only. Older bind tests stub the object's GET with a `[[Server]]` block instead; do not copy that, the fake already models the route. |
| `invariant/configs/<singular>.yml.tmpl` | The generic no-drift, delete-idempotent and destroy-idempotent suites run your config. Add `-init.sh` and `-cleanup.sh` next to it if the resource needs a parent object the harness does not provide. | Register the file name under `EnvMatrix.INPUT_CONFIG` in `acceptance/bundle/invariant/test.toml`. Add an `EnvMatrixExclude` entry for the `DMS=true` leg if the type is not in the deployment-history allowlist, and for cloud if the config cannot run there. Add the type to the exclusion lists in `invariant/migrate` and `invariant/auto-migrate`, which seed state with an older engine. |
| `python/<plural>-support/` | The PyDABs model loads from YAML and from Python and a mutator runs. Six files; copy the newest sibling directory. | `EnvMatrix.PYDAB_VERSION = ["current"]` for a brand-new type. Only for types PyDABs generates (GA or public preview). |
| `refschema/output/<plural>.txt` | Generated listing of the type's fields and which are state. | Regenerated by `-update`; commit it. |

Script conventions, all enforced by review: wrap every command in `trace`; one-line `title` per
phase; `musterr` for a command that must fail; `print_requests.py //<api-path>` instead of inline
`jq` on the request log; `replace_ids.py` right after the first deploy; project any API read you
print to the fields you assert; commit static inputs instead of generating them in the script.

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

Mechanics: one file `libs/testserver/<plural>.go`, a map on `FakeWorkspace` keyed by id, every
handler including delete starts with `defer s.LockUnlock()()`, routes registered in `handlers.go`
next to the neighbouring service. For an asynchronous API, the fake returns a pending operation at
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
go test ./acceptance -run 'TestAccept/bundle/invariant/no_drift' -v -count=1 2>&1 | grep 'INPUT_CONFIG=<singular>'

# the DMS leg runs (or is deliberately excluded)
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -v -count=1 2>&1 | grep -c 'DMS=true'

# effective settings after inheritance
cat acceptance/bundle/resources/<plural>/*/out.test.toml
```

A test that does not appear in the first listing was not selected. Tests under `Cloud = true` also
run locally; the cloud run is additional. On the PR, the Integration Tests check links to the cloud
run; its job log contains a line `Selected N changed tests (...)` followed by `PASS` and `SKIP`
lines with reasons. Confirm your `lifecycle` test is in the `PASS` lines, and quote that in the PR.
Ask a maintainer for the log if you cannot open the run.

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

A new type needs at least one test executed against a real workspace before the PR is ready.
Locally:

```bash
deco env run -i -n aws-prod-ucws -- go test ./acceptance -run 'TestAccept/bundle/resources/<plural>/lifecycle' -v -count=1
```

In the PR description, under Tests, state which directories are `Cloud = true`, which environment
they ran on, and link the run. If the cloud run did not happen, say so and why; do not describe the
local run as cloud coverage.

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
