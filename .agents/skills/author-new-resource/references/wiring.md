# Wiring checklist

`bundle/config/resources/README.md`, section "Adding a resource type", lists every file a new
resource type needs, in order, with the test that fails while it is missing. Read it first. This
reference adds how to decide each declaration, what to check in the generated files, and the rows
no test guards, which are where resource PRs most often come back from review.

Every per-type file is named after the key under `resources:` (`<plural>`, as in `feature_specs`).
`<singular>` is the singular name. `git ls-files '*feature_spec*'` lists a complete example.

## 1. Files to add

- **Config:** the field in `bundle/config/resources.go`; `bundle/config/resources/<plural>.go`
  with the declarations of section 3, and `<plural>_bind_test.go`;
  `bundle/config/mutator/resourcemutator/<plural>_target_mode_test.go`.
- **Schema annotations:** `bundle/internal/schema/annotations/<plural>.yml` (section 4).
- **Direct engine:** `bundle/direct/dresources/<plural>.go`, `<plural>_test.go`, and
  `configs/<plural>.yml` if the type needs hand-written rules (section 2).
- **Fake server:** `libs/testserver/<api>.go` (`testing.md` section 3).
- **Acceptance:** under `acceptance/bundle/`: `resources/<plural>/`, `deployment/bind/<plural>/`,
  `invariant/configs/<plural>.yml.tmpl` and its sidecars, `python/<plural>-support/` when PyDABs
  generates the type (`testing.md` section 2).
- **Generated files** (section 4) and `.nextchanges/bundles/<plural>.md`. Behaviour that needs the
  whole bundle (a mutator, a runner, a pre-deploy check) only if the type needs it, as in the
  README table of that name.

The field is the only shared hand-written file a new type always edits; every other list of
resource types derives from it. Other shared files change only as the outcome of a decision:
`dmsResourceTypes` (section 3), or an opt-out entry with its reason in a coverage test
(`LackingResourceTests`, `LackingBindTests`, `LackingInvariantTest`, `notDeclaring`). Any other
shared list or branch on the type's name means a missed declaration.

## 2. Direct engine rules

The adapter's contract is `bundle/direct/dresources/README.md` and `adapter.go`.

| Add | Where | Caught by |
|---|---|---|
| The input config with every field populated (`registerTestConfig`, or `registerTestDeps` if a parent object must exist first), for the type and its `.permissions` or `.grants`; known type gaps with `registerKnownMissingInStateType`/`registerKnownMissingInRemoteType`, each with a comment | `init()` of `bundle/direct/dresources/<plural>_test.go` | `TestEveryAdapterHasTestFixture` checks that a fixture exists, not that it is complete (section 6) |
| Only the rules the proto cannot express: `provided_id_fields` for identity components, `stable_output_fields` for a server-assigned `name`, `backend_defaults` for server-filled defaults, and a commented `recreate_on_changes` or `ignore_remote_changes` only as a bridge until the upstream annotation lands | `configs/<plural>.yml` (omit if empty) | `TestConfigFiles`, `TestResourceConfig`, `TestResourcesYMLNoRedundantRules`, `TestNameStableOrOptedOut`, `TestNoUpdateResourcesCoverAllFields` (no `DoUpdate`). A missing rule fails only the acceptance case that needs it (`uppercase-name` for `provided_id_fields`) |
| `KeyedSlices()` for repeated fields the API returns unordered, keyed by one real field of the element | adapter file | Nothing checks the key (section 6); the readiness gate decides whether a key exists |
| `api_type: <pkg>.<Type>` only when `apitypes.generated.yml` picked a request type instead of the resource type (`null` for no generated rules) | `configs/<plural>.yml` | Nothing checks the name (section 6) |

## 3. Decisions

Each row is a decision, not a default. The code records it: the declaration in
`bundle/config/resources/<plural>.go`, the registration or exclusion line, with a one-line comment
when the reason is not obvious. The README says which test requires which declaration.

| Decision | Declare | How to decide |
|---|---|---|
| `run_as` | `RunAsSupport()` | Does deploying as another principal make sense for this API? `RunAsApplied` (with `ApplyRunAs`) for resources that run as an identity, `RunAsRejected` when the API takes no identity but the deploying user, `RunAsNotApplicable` otherwise (`run_as.go`) |
| Development-mode presets | `ApplyPresets()`, or `NamePrefixExemptReason()` | Name prefix, tags, paused schedules. Check the API's name length and character constraints against the `[dev user]` prefix before deciding to prefix; the normalized `dev_<user>_` form is the alternative (`model_serving_endpoints.go`). Most Unity Catalog types return `unityCatalogNamePrefixExemptReason` |
| Permissions | a `Permissions` field and `PermissionsSupport()` | Only if the API exposes a public permissions surface; map `CAN_MANAGE`, `CAN_VIEW`, `CAN_RUN` to its levels. The invariant config then attaches permissions |
| Grants | a `Grants` field and `GrantsSupport()` | Only for a Unity Catalog securable; `SecurableType` is its type in grants requests. The invariant config then attaches grants |
| UC dependencies | `UCNames()` | If config names a catalog or schema that may be a bundle resource, point at those fields with the helpers in `uc_names.go` and add `acceptance/bundle/resource_deps/implicit_deps_<singular>/`. If the API takes UUIDs instead of names, there is nothing to capture; document that users reference `${resources.schemas.<key>.schema_id}` explicitly |
| Approval prompts | `Approval()` | Messages when deletion or recreation loses data, is expensive to rebuild, drops child objects, or loses an identity the API minted (a service principal, grants on the old object), with an unused `Order` (`rg -n 'Order:' bundle/config/resources`). Deploy and destroy both, unless deletion is harmless while recreation is not, or destroy's message depends on state (`DestroyFromState`, hook in `bundle/phases/<plural>_approval.go`); say which in a comment. `Approval{}` when nothing is lost. The `basic` golden shows the prompt |
| Workspace URL | `WorkspaceURL()`, and `InitializeURL()` if there is a page | The UI path for one instance, or `false` with a comment saying the UI has no page |
| Launch stage | `SchemaLaunchStage()`, or nothing | Nothing means a public Python package. `PUBLIC_BETA` or `PRIVATE_PREVIEW` only for an intentional difference in bundle availability, as the Postgres types do; PyDABs then skips the type (readiness check 2.10) |
| Bind | `registerBindFixture` or `registerNotBindable` in `<plural>_bind_test.go` | The fixture's `id` (`"0"`, or a composed id) and a mock for the Get call `Exists` makes; not bindable only with the reason |
| Create recovery | `retrySafe`, `recoverCreate`, or `registerCreateRecoveryOptOut` | See "Failed create requests" in `bundle/direct/dresources/README.md` |
| Deployment history | `dmsResourceTypes` in `libs/testserver/bundledeployments.go` (shared), or an exclusion | The list mirrors the deployment-history service's allowlist: add the type only with that backend change, coordinated. Otherwise exclude the `DMS=true` leg with a comment in `resources/<plural>/test.toml` and the invariant `-exclude.toml`; the leg fails on the first deploy until you do |
| Etag protection, when edits since the last deployment must be preserved | `etag_field` in `configs/<plural>.yml`, `LoadDeployedState()`, a check registered with `preDeployChecks.Register` | Dashboards are the precedent (`bundle/deploy/check_dashboards_modified_remotely.go`); readiness check 2.11 describes the flow |

Automatic, from `BaseResource` and the derived lists: `lifecycle.prevent_destroy`,
`lifecycle.depends_on`, `bundle summary`, `bundle open` (refuses when `WorkspaceURL()` declares no
page), `databricks experimental open`, bind and unbind through `Exists`, state loading.

## 4. Generated files

Planning rules and the user-facing schema are separate. `configs/<plural>.yml` controls plan
classification; `ignore_remote_changes` does not hide a field from JSON schema or Python models.
Before adding a resource-specific schema transform, check `annotation.Descriptor.OutputOnly`: a
hand-written annotations file accepts `x-databricks-field-behaviors_output_only: true` under a
field. The existing `addAnnotations` and `removeOutputOnlyFields` passes then hide it from authored
input without changing the SDK or proto. An `IDENTIFIER` field can legitimately remain in the API
even when the bundle exposes a separate create-time ID. Do not assume a `bundle:"readonly"` shadow
hides an embedded SDK field: `fromTypeStruct` skips tagged fields before deduplicating property
names, so check the output.

Run in this order after the Go types compile; `./task generate` runs all of them.

| Command | Writes | Check |
|---|---|---|
| `./task generate-refschema` | `acceptance/bundle/refschema/output/<plural>.txt` | File exists and lists the resource's fields with `STATE` or `ALL` |
| `./task generate-schema` | `bundle/schema/jsonschema.json`, and `bundle/internal/schema/annotations/<plural>.generated.yml` with a `PLACEHOLDER` per undocumented field | Document each placeholder at the same tree position in `annotations/<plural>.yml` (entries for upstream fields move to `<plural>.overrides.yml` on regenerate) and run again until `<plural>.generated.yml` is gone; `TestRequiredAnnotationsForNewFields` fails on leftovers. See the README in that directory |
| `./task generate-validation` | `bundle/internal/validation/generated/required_fields.go`, `enum_fields.go` | Entries for the type appear |
| `./task generate-direct` | `bundle/direct/dresources/apitypes.generated.yml`, `configs/<plural>.generated.yml` | The plural has an entry naming the resource type; the generated rules match the proto's `IMMUTABLE`, `OUTPUT_ONLY`, `INPUT_ONLY`. A missing entry means the heuristic found nothing: set `api_type` |
| `./task pydabs-codegen` | `python/databricks/bundles/<plural>/`, `core/`, docs, test cases | Produced unless `SchemaLaunchStage()` is `PRIVATE_PREVIEW` or `PUBLIC_BETA`. Regenerate the public API snapshot: `cd python && UPDATE_SNAPSHOTS=1 uv run pytest databricks_tests/core/test_public_api.py`. The task deletes the generated packages before it fetches dependencies; if it fails on network access, restore with `git checkout -- python/`, sync the `python/` and `python/codegen/` environments first, and rerun |
| `go test ./acceptance -run 'TestAccept/experimental/open' -update`, `go test ./acceptance -run '^TestAccept$' -only-out-test-toml` | `acceptance/experimental/open/output.txt` and the invariant `out.test.toml` files, which list every type | The type appears |
| `./task generate-check`, then `git status` | the reproducible generators again | No diff; CI runs it followed by `git diff --exit-code` |

Never edit `jsonschema.json`, `*.generated.yml`, the validation files or the Python packages by
hand.

## 5. Reading the generated rules back

After `./task generate-direct`, open `configs/<plural>.generated.yml` and compare with the API docs:

- Every field GET does not return should be under `ignore_remote_changes` with `spec:input_only`.
- Every server-computed field should be there with `spec:output_only`.
- Every create-only field should be under `recreate_on_changes` with `spec:immutable`.

Anything missing from that list and present in your hand-written `configs/<plural>.yml` is an
annotation the proto lacks. File it upstream and keep the hand rule commented as a bridge.

If the generated file is absent or says there are no field behaviours for the resource, the
generator did not find a matching API type: either the config type is a hand-written mirror (embed
the SDK type instead) or the heuristic picked nothing (set `api_type`). Do not predict generated
rules in a plan; read them back after `./task generate-direct`.

## 6. What no test guards

Every other row of this checklist has a test that fails while the row is missing. Verify these by
hand.

| Row | Verify |
|---|---|
| The acceptance set; `TestResourceTestDirsCoverage` requires `resources/<plural>/` to exist, not what it holds | `ls acceptance/bundle/resources/<plural>/` lists `basic`, `lifecycle`, `remote-delete`; `grep Cloud acceptance/bundle/resources/<plural>/lifecycle/out.test.toml` shows `Cloud = true` |
| The test fixture populates every field | Compare the `registerTestConfig` call in `<plural>_test.go` with the input fields in `acceptance/bundle/refschema/output/<plural>.txt` |
| The `KeyedSlices` key is a real, client-settable field | Readiness check 2.2, and an acceptance case that reorders the list and plans nothing |
| `api_type` names an existing type; a misspelled one yields no rules | `configs/<plural>.generated.yml` exists and lists the behaviours you expect |
| UC names in a field without `catalog` or `schema` in its name (a three-part `name`, a `parent`), which `TestResourceTypesWithUCNameFieldsDeclareThem` does not see | `rg -n UCNames bundle/config/resources/<plural>.go`; the implicit-deps case deploys the catalog or schema first |
| A development-mode preset shows in a deployment | `rg -l 'mode: development' acceptance/bundle/resources/<plural>` finds a case whose output shows the displayed name |
| Etag protection: warning, refusal, explicit overwrite | An external-edit case warns on plan, refuses ordinary deploy and allows `--force`, like `acceptance/bundle/resources/dashboards/detect-change` |
| The launch stage; only the Postgres types are checked | `python/databricks/bundles/<plural>/` exists exactly when you decided it should |
| The changelog fragment; `./task check-changelog` checks only fragments that exist | `ls .nextchanges/bundles/<plural>.md` |
| No other shared hand-written file | `git diff --name-only origin/main...HEAD -- . ':!acceptance/bundle' ':!python' ':!*.generated.yml' ':!bundle/schema/jsonschema.json' ':!bundle/internal/validation/generated' ':!.nextchanges'` lists only files named after the type, the fake server's file, `bundle/config/resources.go` and the entries of section 1 |
