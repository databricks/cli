# Wiring checklist

Every place a new resource type is registered, generated or tested. For each row: what to add, and
what catches it when forgotten. Rows marked **nothing catches it** are the ones to verify by hand
with the command given; they are where resource PRs most often come back from review.

Use `<singular>` and `<plural>` for the resource names, `<Type>` for the Go identifier.

## 1. Config type and registration (`bundle/config`)

| Add | Where | Caught by |
|---|---|---|
| `type <Type> struct { BaseResource; ID string \`json:"id,omitempty" bundle:"readonly"\`; <sdk>.<Resource> }` with value-receiver `MarshalJSON` and pointer-receiver `UnmarshalJSON` via the SDK `marshal` package | `bundle/config/resources/<singular>.go` (new) | `TestCustomMarshallerIsImplemented` in `bundle/config/resources_test.go` |
| `Exists(ctx, client, id)` using the service's Get and `apierr.IsMissing` | same file | `TestResourcesBindSupport`: add an instance to its fixture and a mock expectation for the Get call. The test binds with the id `"0"`; a composed id needs an entry in its per-type id map |
| `ResourceDescription()` with singular, plural and titles | same file | Empty plural panics at startup in `collectResourceMap` |
| `GetName()`, `InitializeURL(baseURL)`, `GetURL()` | same file | `TestBundleResourcePluralNamesResolveInWorkspaceURLs`: add a URL pattern (row below) or add the plural to that test's no-URL map with a comment saying the UI has no page |
| Map field `<Plural> map[string]*resources.<Type> \`json:"<plural>,omitempty"\`` | `bundle/config/resources.go`, the `Resources` struct | Everything below keys off it |
| Entry in `AllResources()` | `bundle/config/resources.go` | `TestResourcesAllResourcesCompleteness` |
| Entry in `SupportedResources()` | `bundle/config/resources.go` | `TestSupportedResources` |
| `Grants` or `Permissions` field on the type only if the API exposes a public grants or permissions surface for it | `bundle/config/resources/<singular>.go` | Derives `<plural>.grants` or `<plural>.permissions` keys that then need rows in sections 2 and 6 |

## 2. Direct engine adapter (`bundle/direct/dresources`)

| Add | Where | Caught by |
|---|---|---|
| `Resource<Type>` with `New`, `PrepareState`, `DoRead`, `DoCreate`, `DoUpdate` (only if the API can update), `DoDelete`, and `WaitAfterCreate`/`WaitAfterUpdate`/`WaitAfterDelete` for asynchronous APIs. Helper methods unexported; one SDK call per method; SDK errors returned as they are | `bundle/direct/dresources/<singular>.go` (new) | `NewAdapter` rejects wrong signatures and extra exported methods; `TestAll` runs CRUD against the fake |
| Entry `"<plural>": (*Resource<Type>)(nil)` | `bundle/direct/dresources/all.go` | A config type without an adapter fails only at plan time with "resource type not supported". **Nothing catches it in unit tests**; verify with `go test ./bundle/direct/dresources -run 'TestAll/^<plural>$' -v -count=1` |
| `testConfig["<plural>"]` with every field populated; `testDeps` if a parent object must exist first | `bundle/direct/dresources/all_test.go` | A missing entry runs `TestAll` with a zero config and proves nothing. **Nothing catches it**; check the entry exists |
| `configs/<plural>.yml` with only the rules the proto cannot express: `provided_id_fields` for identity components, `stable_output_fields` for a server-assigned `name`, `backend_defaults` for server-filled defaults, and a commented `recreate_on_changes` or `ignore_remote_changes` only as a bridge until the upstream annotation lands | `bundle/direct/dresources/configs/<plural>.yml` (omit if empty) | `TestConfigFiles`, `TestResourceConfig`, `TestResourcesYMLNoRedundantRules` (a hand rule the generated file already produces fails), `TestNameStableOrOptedOut`, `TestNoUpdateResourcesCoverAllFields` (types without `DoUpdate`) |
| `KeyedSlices()` for repeated fields the API returns unordered, keyed by one real field of the element | adapter file | **Nothing catches a wrong key**; the readiness gate decides whether a key exists |
| `apitypes.yml` pin only when `apitypes.generated.yml` picked a request type instead of the resource type | `bundle/direct/dresources/apitypes.yml` | **Nothing catches a wrong or misspelled pin**; a typo yields an empty generated config. Verify `configs/<plural>.generated.yml` lists the behaviours you expect |

## 3. Generated files

Run in this order after the Go types compile; `./task generate` runs all of them.

| Command | Writes | Check |
|---|---|---|
| `./task generate-refschema` | `acceptance/bundle/refschema/output/<plural>.txt` | File exists and lists the resource's fields with `STATE` or `ALL` |
| `./task generate-schema` | `bundle/schema/jsonschema.json`, and rewrites `bundle/internal/schema/annotations.yml` adding `PLACEHOLDER` entries | Fill every placeholder for the type (resource-level description, `lifecycle`, each field without an upstream description), then run again. `TestRequiredAnnotationsForNewFields` in `bundle/internal/schema` fails on leftovers |
| `./task generate-validation` | `bundle/internal/validation/generated/required_fields.go`, `enum_fields.go` | Entries for the type appear |
| `./task generate-direct` | `bundle/direct/dresources/apitypes.generated.yml`, `configs/<plural>.generated.yml` | The plural has an entry naming the resource type; the generated rules match the proto's `IMMUTABLE`, `OUTPUT_ONLY`, `INPUT_ONLY`. A missing entry means the heuristic found nothing: pin it |
| `./task pydabs-codegen` | `python/databricks/bundles/<plural>/`, `core/_generated/`, docs, test cases | Produced for every resource except those with a `PRIVATE_PREVIEW` or `PUBLIC_BETA` entry in `launchStageOverrides` (`bundle/internal/annotation/preview.go`). Regenerate the public API snapshot: `cd python && UPDATE_SNAPSHOTS=1 uv run pytest databricks_tests/core/test_public_api.py`. The task deletes the generated packages before it fetches dependencies; if it fails on network access, restore with `git checkout -- python/`, sync the `python/` and `python/codegen/` environments first, and rerun |
| `./task generate-check` | nothing | Must be clean; CI runs it |

Never edit `jsonschema.json`, `*.generated.yml`, the validation files or the Python packages by
hand. `annotations.yml` is the one hybrid file: generated structure, hand-written text.

## 4. Fake server (`libs/testserver`)

| Add | Where | Caught by |
|---|---|---|
| `<Plural> map[string]<sdk>.<Resource>` on `FakeWorkspace`, initialised in `NewFakeWorkspace` | `libs/testserver/fake_workspace.go` | `TestAll` 404s |
| Create, get, update, delete handlers; `MapGet`/`MapDelete` where the stored body is the request body; a dedicated method where the server assigns fields or applies a mask | `libs/testserver/<plural>.go` (new), routes in `handlers.go` next to the neighbouring service | `TestAll` and every acceptance test |
| The plural in `dmsResourceTypes`, mirroring the deployment-history service's allowlist (a backend change you coordinate), or an `EnvMatrixExclude` for `DMS=true` with a comment in the resource's `test.toml` and in `acceptance/bundle/invariant/test.toml` | `libs/testserver/bundledeployments.go` | **Nothing catches it in Go**; the `DMS=true` acceptance leg fails on the first deploy |

## 5. Bundle-wide behaviour (`bundle/config/mutator/resourcemutator`, `bundle/phases`, `libs/workspaceurls`)

Each row is a decision, not a default. The code records it: the entry in the test's list, the
approval table, the exclusion line, with a one-line comment when the reason is not obvious.

| Decision | Where | Caught by |
|---|---|---|
| `run_as`: does deploying as another principal make sense? If not, add a "not supported" block | `run_as.go`; test lists in `run_as_test.go` (`allResourceTypes` exact list, `allowList`) | `TestRunAs*` fail until the type is in the exact list and either allowed or blocked |
| Development-mode presets: name prefix, tags, paused schedules. Check the API's name length and character constraints against the `[dev user]` prefix before deciding to prefix | `apply_presets.go`; fixture and carve-out list in `apply_target_mode_test.go` | `TestAllResourcesMocked`, `TestAppropriateResourcesAreRenamed` (a user-facing `Name` must be prefixed or carved out with a reason); the preset itself needs an acceptance test, nothing enforces that |
| Permissions: level map entry if the API has permissions, else the unsupported list | `apply_bundle_permissions.go`, `apply_bundle_permissions_test.go` | `TestAllResourcesExplicitlyDefinedForPermissionsSupport` |
| UC dependencies: if config names a catalog or schema by name that may be a bundle resource, rewrite it to `${resources.schemas...}`. If the API takes UUIDs instead of names, there is nothing to capture; document that users reference `${resources.schemas.<key>.schema_id}` explicitly | `capture_uc_dependencies.go` | **Nothing catches it**; add a case to the implicit-deps acceptance test |
| Approval prompt for deletion or recreation that loses data, is expensive to rebuild, drops child objects, or loses an identity the API minted (a service principal, grants on the old object) | The approval table in `bundle/phases` (older trees: separate lists in `deploy.go` and `destroy.go`), message in `messages.go`. Deploy and destroy both, unless destroy's prompt comes from somewhere else or deletion is harmless while recreation is not; say which in a comment on the entry | **Nothing catches a missing entry**; the resource's `basic` golden shows the prompt |
| Workspace URL: pattern if the UI has a page | `libs/workspaceurls/urls.go`; `cmd/experimental/workspace_open_test.go` lists; `acceptance/experimental/open/output.txt` | `TestBundleResourcePluralNamesResolveInWorkspaceURLs`; the `workspace_open_test.go` lists are exact and fail until updated |
| State fixtures | `bundle/statemgmt/state_load_test.go`: one entry per `TestStateToBundle*` fixture | The tests assert full resource coverage and fail for a type without a fixture |
| `bundle run`: only if the resource has a run action | `bundle/run/runner.go` | Not applicable to most types |

Automatic through `BaseResource`: `lifecycle.prevent_destroy`, `lifecycle.depends_on`, telemetry,
`bundle summary`, `bundle open` (refuses when `GetURL` returns false), bind and unbind through
`Exists`.

## 6. Tests and fixtures

See `testing.md` for the content of each. Registration points only:

| Add | Where | Caught by |
|---|---|---|
| `acceptance/bundle/resources/<plural>/{basic,lifecycle,remote-delete}/` | new directories | Documented requirement, **not enforced**; reviewers check |
| `acceptance/bundle/deployment/bind/<singular>/` | new directory | same |
| `acceptance/bundle/invariant/configs/<singular>.yml.tmpl` (+ `-init.sh`, `-cleanup.sh` if needed) | new files | `TestInvariantConfigsCoverage` checks the file exists |
| The file name under `EnvMatrix.INPUT_CONFIG`, plus `EnvMatrixExclude` for `DMS=true` or cloud when needed, plus exclusions in `invariant/migrate/test.toml` and `invariant/auto-migrate/test.toml`; `invariant/continue_293` seeds every config with an old CLI that does not know the type, so exclude it there if that leg fails | `acceptance/bundle/invariant/test.toml` and the subdirectories | **Nothing catches a missing matrix entry**; verify with `go test ./acceptance -run 'TestAccept/bundle/invariant/no_drift' -v -count=1 2>&1 \| grep 'INPUT_CONFIG=<singular>'` |
| `acceptance/bundle/python/<plural>-support/` when PyDABs generates the type | new directory, six files | `test_python_support_coverage` |
| `.nextchanges/bundles/<plural>.md` | new file | `task check-changelog` once the PR exists; reviewers otherwise. The line ends with the PR link, so fill the number in right after opening the PR |

## 7. Reading the generated rules back

After `./task generate-direct`, open `configs/<plural>.generated.yml` and compare with the API docs:

- Every field GET does not return should be under `ignore_remote_changes` with `spec:input_only`.
- Every server-computed field should be there with `spec:output_only`.
- Every create-only field should be under `recreate_on_changes` with `spec:immutable`.

Anything missing from that list and present in your hand-written `configs/<plural>.yml` is an
annotation the proto lacks. File it upstream and keep the hand rule commented as a bridge.

If the generated file is absent or says there are no field behaviours for the resource, the
generator did not find a matching API type: either the config type is a hand-written mirror (embed
the SDK type instead) or the heuristic picked nothing (pin the type in `apitypes.yml`). Do not
predict generated rules in a plan; read them back after `./task generate-direct`.
