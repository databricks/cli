---
name: onboard-bundle-resource
description: Add a Databricks resource type to Declarative Automation Bundles, including its direct adapter, schema, Python support, bundle behavior, and lifecycle tests. Use for onboarding or completing support for a resources section; generating workspace API commands alone is a separate workflow.
---

# Onboard a bundle resource

Work from the repo root. Read [the adapter contract](../../../bundle/direct/dresources/README.md) before implementing a resource, and follow [generated-file ownership](../../rules/auto-generated-files.md) and [testing rules](../../rules/testing.md).

Use `feature.go` for a small recent resource and `model_service.go` for an AIP resource with separate create identity, a nested body, and UC grants. Both have config wrappers and direct adapters. Read [examples and review lessons](references/precedents.md) when choosing a pattern or investigating a special API contract. The target API determines which parts of an example apply.

## Establish the contract

Inspect the SDK version in `go.mod`, its service models/methods, and the corresponding entries in `.codegen/cli.json`. Establish these decisions before choosing an adapter shape; document the non-obvious ones alongside the implementation:

- Bundle plural, SDK resource/body type, create request type, get response type, and API launch stage.
- User-authored identity (`parent`, leaf ID, full name), API resource name, persisted deployment ID, bind input, and UI URL. These may differ; keep conversions consistent across create, read, exists, delete, bind, and summary.
- Mutable, immutable/create-only, input-only, output-only, and defaulted fields; required and oneof constraints; clearing semantics for absent values, `false`, `0`, empty strings and collections.
- Update support and accepted mask paths; asynchronous completion; missing-resource deletion; permissions/grants; execution identity and presets.

Use API docs and upstream proto annotations to resolve ambiguity. If `~/universe` is available, start with `deco/api/tech-docs/api-resources.md`, locate the service proto, and inspect its resource, field-behavior, method and HTTP annotations. The rules under `deco/api/health/linter` explain Databricks-specific AIP requirements. Upstream can be ahead of the pinned SDK/spec: do not expose unreleased fields merely because they exist there. Without that checkout, use the pinned spec, SDK, and published API docs.

If SDK/spec support is missing, identify that prerequisite and keep its bump separate from resource onboarding; use the repo's `bump-sdk` skill when needed. `.codegen/cli.json` and SDK models are generated inputs: correct upstream annotations and regenerate through their owning workflow. A pending upstream fix may need a narrowly documented local rule.

## Wire the resource

| Location | Work |
| --- | --- |
| `bundle/config/resources/<singular>.go` | Define the config wrapper with `BaseResource`, read-only `ID`, resource description, `Exists`, `GetName`, and URL initialization. |
| `bundle/config/resources.go` | Add the map field, `AllResources()` collection, and `SupportedResources()` description. `ResourcesTypes` is derived; do not create a parallel registry. |
| `bundle/direct/dresources/<singular>.go` | Implement the adapter using the established types and API semantics. |
| `bundle/direct/dresources/all.go` | Register the adapter under the same plural. Register supported grants/permissions sub-resources separately. |
| `bundle/direct/dresources/configs/<plural>.yml` | Add lifecycle/diff rules that generation does not provide, with reasons tied to the API contract. Omit an empty file. |
| `libs/workspaceurls/urls.go` | Add the real UI route and test it; summary should have a useful name and URL. |
| `libs/testserver/` | Add stateful service handlers, routes in `handlers.go`, and storage in `FakeWorkspace` as needed. |

Define value-receiver `MarshalJSON` and pointer-receiver `UnmarshalJSON` on the outer config wrapper using SDK `marshal`. An embedded SDK marshaler can otherwise discard wrapper fields. Handwritten config structs with optional scalar fields need their own `ForceSendFields` and marshal methods. `TestResourceTypesZeroValueFieldsSerialize` guards serialization and ForceSendFields routing; request or round-trip assertions must also verify that explicit zeros reach the payload.

SDK request fields tagged `json:"-"` are invisible to the bundle schema. If create identity lives outside the body, use a bundle-local config with JSON-visible identity plus mutable body fields, as `ModelServiceConfig` does. Keep lifecycle, permissions, grants, and other wrapper-only fields out of `PrepareState` unless they belong to the resource's API state.

If there is no UI page, override `GetURL` to return `"", false` and document the exception in `TestBundleResourcePluralNamesResolveInWorkspaceURLs` in `bundle/config/resources_test.go`. Do not invent a URL or add an exception just to pass the test. `GetName` must work from authored config; an output-only API name may not be populated there.

## Implement predictable deployment

The adapter README owns the detailed method contract and field-rule definitions. Pay particular attention to these onboarding decisions:

- Prefer one SDK mutation per `DoCreate`/`DoUpdate`/`DoDelete`. Return SDK errors directly here: the framework adds operation, resource key and ID context. Put asynchronous readiness in `WaitAfterCreate`/`WaitAfterUpdate` so state is persisted before polling. Use existing SDK/retry helpers and honor the passed context. Add `WaitAfterDelete` when asynchronous teardown would race recreation.
- Send the desired full mutable state on update. Prefer `update_mask: "*"` when supported; otherwise use a documented static list of API field paths. Do not derive masks from `entry.Changes`, which contains engine paths. Test field removal and zero-value serialization in any newly constructed request body.
- Classify immutable identity and create-only fields during planning. If there is no update operation, classify every settable field so none can plan an unsupported update; do not implement a successful no-op `DoUpdate`. `TestNoUpdateResourcesCoverAllFields` checks this.
- Normalize differently shaped reads in `DoRead`. Its return type defines the resource's reference schema: retain outputs users need to reference, such as the canonical name, even when absent from desired state. Keep `RemapState` a trivial subset copy: synthesizing fields there can suppress real drift as `missing_in_remote`. Preserve remote values and express suppression in YAML. Effective/defaulted output does not necessarily represent the user's original input.
- For backend-assigned names stable across an in-place update, declare `stable_output_fields` and test a dependent resource referencing that name. `TestNameStableOrOptedOut` prevents unnecessary child recreation. Do not pin mutable user-authored names stable.
- Check unordered repeated fields for `KeyedSlices`. Declare `provided_id_fields` for user inputs composing an immutable name-based ID; local edits recreate while remote normalization is tolerated. An in-place rename uses `updatable_id_fields` and requires `DoUpdateWithID` support.

For singletons, determine whether create means update/upsert and delete means reset, and which identity is bindable. A missing create API is not a reason to copy unrelated CRUD. Older Postgres inline polling and change-derived masks encode particular backend constraints and are not templates for new adapters.

## Check bundle-wide behavior

Decide each item explicitly; do not add every resource to an unsupported list by default.

| Behavior | Where to check |
| --- | --- |
| `run_as` | `bundle/config/mutator/resourcemutator/run_as.go` and its tests. Propagate supported execution identity; reject incompatible execution behavior with an actionable diagnostic. Passive resources can coexist with a different run identity without a new rejection. Update coverage lists to reflect the decision. |
| Presets/development mode | `apply_presets.go` and the exhaustive fixture in `apply_target_mode_test.go` in the same directory. Decide prefix, tags and other applicable presets; keep hierarchy/name formats valid. Cover changed behavior in acceptance tests. |
| UC dependencies | `capture_uc_dependencies.go` in the same directory. Capture parent catalog/schema dependencies, including APIs whose parent is a resource-name string. |
| Permissions/UC grants | Resource wrapper, `apply_bundle_permissions.go` and its coverage tests, direct `permissions.go`/`grants.go`, and `all.go`. Use shared sub-resource machinery. Verify a public API supports the securable type. Shared grants pass the parent's `.id` as UC `FullName`; check that contract when choosing the deployment ID. |
| Bundle launch stage | `bundle/internal/annotation/preview.go` owns CLI launch-stage overrides. Check whether a bundle-local config retains its upstream stage and whether the bundle release stage differs. Add an override when needed and verify the generated schema, particularly for handwritten singleton types. |
| State loading | `bundle/statemgmt/state_load_test.go` has exhaustive fixtures/assertions for IDs and modified status. A newly introduced type needs coverage; changing an already-shipped persisted shape also requires compatibility handling in `bundle/direct/dstate/migrate.go`. |
| Bind/unbind | `Exists`, `bundle/config/resources_test.go` bind coverage, and `acceptance/bundle/deployment/bind/<singular>/`. Test the documented identity format and subsequent deploy/destroy. |
| Deployment metadata service (DMS) | `libs/testserver/bundledeployments.go` lists supported prefixes and the upstream files owning them. Verify service support/rollout; a fake-only registration cannot enable deployment history. |

If DMS does not support the type yet, identify the upstream dependency and report the limitation. Keep the fake faithful to the service. An intentional temporary matrix exclusion needs an explicit reason in the resource and invariant tests; it does not establish DMS support.

## Generate and inspect

After registering the Go types, run generators in dependency order:

```bash
./task generate-refschema
./task generate-schema
./task generate-validation
./task generate-direct
./task pydabs-codegen
```

`./task generate` runs these in order and also regenerates CLI commands from the checked-in spec. It does not refresh the spec from upstream.

Inspect the resource in the generated outputs:

- `acceptance/bundle/refschema/out.fields.txt`: identity and nested fields must be visible.
- `bundle/direct/dresources/apitypes.generated.yml`: verify the selected API type, not just the plural's presence. Field-overlap discovery can select a request or parent type with wrong field behaviors. Pin the correct type in handwritten `apitypes.yml` with a reason when needed; scrutinize singletons and bundle-local configs.
- `configs/<plural>.generated.yml`: check output-only, immutable and input-only behavior against the API. Empty generated files are legitimately omitted; missing annotations do not imply mutability. Fix source mappings/annotations instead of duplicating generated rules by hand.
- `bundle/internal/schema/annotations.yml` and `bundle/schema/jsonschema.json`: `generate-schema` imports upstream docs and syncs annotation entries. Do not paste API docs into annotations by hand. Fill CLI-owned descriptions or deliberate overrides/placeholders there, then regenerate. The docs-only schema is published on the release `docgen` branch, not maintained on main.
- `python/databricks/bundles/<plural>/`, generated core wiring/tests, and Python docs: discovery is automatic from the bundle schema via `python/codegen/codegen/packages.py`. GA/public-preview resources are eligible; `PUBLIC_BETA` and `PRIVATE_PREVIEW` are excluded. Verify the generated schema's actual launch stage. Do not add a second manual registry or denylist a resource to hide a generation bug.

Review intentional Python public API changes and update the snapshot with `cd python && UPDATE_SNAPSHOTS=1 uv run pytest databricks_tests/core/test_public_api.py`. Re-run without `UPDATE_SNAPSHOTS`. Do not hand-edit generated Python or schema outputs.

## Prove the lifecycle

Use existing test harnesses and helpers:

1. Add the adapter fixture and prerequisites to `bundle/direct/dresources/all_test.go` (`testConfig`/`testDeps`). Check config serialization, resource-list completeness, bind support, URL coverage, state loading, and mutator coverage lists.
2. Add focused tests in `acceptance/bundle/resources/<plural>/`. Cover deploy, plan/debug plan, summary with ID/URL, mutable update, field clearing, immutable recreation, remote deletion, and destroy as applicable. **Execute** immutable changes and assert delete/create requests; plan-only checks miss apply bugs. Verify no drift after create and update. Model real error/default/mask behavior in the shared test server.
3. Read `acceptance/bundle/invariant/README.md`. Add `configs/<singular>.yml.tmpl` **and its `EnvMatrix.INPUT_CONFIG` entry in `acceptance/bundle/invariant/test.toml`**; an unselected fixture never runs. Include supported grants/permissions. Mark removable lines `# CAN_REMOVE` to test clearing; reserve a reasoned `# CANNOT_REMOVE` for demonstrated limitations.
4. Exclude a new direct-only template from historical `invariant/migrate` and `invariant/auto-migrate` matrices with a reason. Check `invariant/continue_293` too: its old CLI cannot seed newly introduced types. These are compatibility boundaries, not broken tests to skip; do not add new Terraform converters or historical fixtures.
5. Add bind/unbind coverage when supported. Add `acceptance/bundle/python/<plural>-support/` for every PyDABs-exposed resource, following its [README](../../../acceptance/bundle/python/README.md). It exercises YAML, Python construction and a mutator; new resources use the current wheel. Python's coverage test enforces its presence.
6. Run local tests, then at least one live-cloud lifecycle test for a new type, as required by the adapter README. `Cloud = true` adds a cloud run; it still runs locally. Use the available cloud-test workflow, provision dependencies and arrange cleanup. Record unavailable environments or unrun coverage accurately.

Useful local commands, replacing `<plural>` with the new resource:

```bash
go test ./bundle/config/... ./bundle/direct/... ./libs/testserver/... ./libs/workspaceurls/...
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>' -update
go test ./acceptance -run 'TestAccept/bundle/resources/<plural>'
./task pydabs-test
```

Also run the selected invariant, bind and Python acceptance tests; inspect verbose test names to confirm the new fixture ran. Regenerate snapshots via `-update`/`./task test-update`, then run without update. Filter API read output to asserted fields and record mutation requests with `print_requests.py`; see the testing rules. Test applicable presets, dependencies and references explicitly.

Before submitting, follow the repo's [PR checklist](../pr-checklist/SKILL.md), including full tests, formatting/lint and generation checks. Add a `.nextchanges/bundles/` fragment for public resource support. Describe the supported lifecycle, API-specific limits, and local/cloud validation; do not claim excluded modes were tested.
