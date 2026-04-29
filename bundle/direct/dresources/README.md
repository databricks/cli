# Guidelines on implementing a resource

## Core constraints

 - See adapter.go on what methods are needed and what constraints are present.
 - Return SDK errors directly, no need to wrap them. Things like current operation, resource key, id are already added by the caller and will be part of the error message.
 - Although the arguments to resource methods like DoCreate are pointers, they are never nil, so nil checks are not needed. The passed id argument is never empty string.
 - When returning id from DoCreate() and from DoUpdateWithID() there is no need to check that returned id is non-empty, this will be done by the framework and converted to error.
   An exception could be made if default error message lacks the necessary context.
 - The arguments point to actual struct that will be persisted in state, any changes to it will affect what is stored in state. Usually there is no need to change it, but if there is, there should always be detailed explanation.
 - Each Create/Update/Delete method should correspond to one API call. We persist state right after, so there is minimum chance of having orphaned resources.
 - We should calculate the update type during plan phase. This means it should be configured via resources.yml as much as possible, falling back to OverrideChangeDesc(). The DoUpdate() implementation should be as predictable as possible based on the plan. In particular, avoid reading remote state in DoUpdate() to decide what kind of update to dod.
 - Create/Update/Delete methods should not need to do read requests. They can read state passed to them via \*PlanEntry but that should be reserved for exceptional cases. Most resources should have 1-1 mapping to single SDK/API call.
 - For update with complex logic, ensure that DoUpdate() never results in no-op. If certain fields could not be updated, they should be excluded at plan level in resources.yml.

## Field classification in resources.yml

Each field with special plan/deploy behavior must be declared in `resources.yml`. Choose the right category:

 - **`backend_defaults`**: The backend may fill in a value when the user doesn't specify one. Suppresses the diff when the user's config is nil/empty but remote has a value. Optionally restrict to specific allowed remote values via `values:`. Use for fields the API fills in as defaults (e.g., `format`, `run_if`, `node_type_id`). Link to TF provider suppression comment in the same format as existing entries.
 - **`ignore_remote_changes`**: Ignore changes the remote makes to this field. Use for fields the backend manages (e.g., cloud-provider attributes like `aws_attributes`, `gcp_attributes`) or fields not returned by the update endpoint. Reason codes:
   - `output_only` — the field is computed by the backend; the user never sets it
   - `input_only` — accepted on create/update but not returned by GET (e.g., write-only tokens, flags)
   - `managed` — managed by the cloud provider or platform, not by the user config
 - **`ignore_local_changes`**: Ignore changes the user makes to this field. Use for fields that cannot be updated via API — either they are immutable after creation or require a separate API that is not yet implemented. Must have a comment in resources.yml explaining why.
 - **`recreate_on_changes`**: Changing this field requires delete + create. Use for truly immutable fields (name, type, location). The reason should reference API docs or TF provider.
 - **`update_id_on_changes`**: Changing this field changes the resource's ID. Requires `DoUpdateWithID` to be implemented.

## Update mask

When implementing DoUpdate, use a **static list** of updatable API field names or `*` if the API supports it.

Do **not** derive update mask field names from `entry.Changes`. The paths in `entry.Changes` are engine-internal Go struct paths, not API field names. Mapping them to API fields is fragile: it breaks when struct layout changes, silently skips nested updates, and conflicts with the direct engine's full-update model.

If a resource has fields that must not be sent in updates (deploy-only, lifecycle-only, etc.), document them explicitly with a `var` block and a comment explaining each exclusion.

## Async APIs: WaitAfterCreate / WaitAfterUpdate

For resources whose create or update is asynchronous (the resource is not immediately ready after the call returns), implement `WaitAfterCreate` and/or `WaitAfterUpdate` instead of polling inline inside DoCreate/DoUpdate. These are the correct extension points in the framework, and polling inline bypasses state persistence timing.

## Slice ordering: KeyedSlices

If the API may return a slice's elements in a different order between calls (e.g., `depends_on` in job tasks, `privileges` in grants), implement `KeyedSlices` to compare elements by a natural key rather than by index. Without this, every deploy after any reordering shows phantom diffs.

```go
func (*ResourceJob) KeyedSlices() map[string]any {
    return map[string]any{
        "tasks":               getTaskKey,
        "tasks[*].depends_on": getDependsOnTaskKey,
    }
}
```

## State backward compatibility

The state struct is serialized to JSON and persisted between deploys. Changing a `json:"..."` tag on a state field — including the `__embed__` convention — is a **backward compatibility break**: existing state files will fail to deserialize.

When you must rename a JSON key, bump the state schema version and add migration logic. Verify with a test that deserializes state written by the old schema.

## OverrideChangeDesc

Use `OverrideChangeDesc` only as a last resort when `resources.yml` settings cannot express the needed logic. Skipping an action with `change.Action = deployplan.Skip` in `OverrideChangeDesc` creates a silent no-op: the plan shows no change even if the user's config differs from remote. Document the skip reason clearly in both the comment and `change.Reason`.

## Nice to have
 - Add link to corresponding API documentation before each method.
 - Add link to corresponding terraform resource implementation at the top of the file.

## Testing

 - Make sure to implement CRUD for testserver in libs/testserver
 - Test first with go test ./bundle/direct/dresources
   - You might need to add test fixture in all\_test.go
 - Implement acceptance tests in acceptance/bundle/resources/<resource>/<subtest>
   - See acceptance/bundle/resources/volumes
   - Prefer smaller tests for each operation.
   - Make sure bundle deploy/plan/debug plan/summary/destroy are covered
 - Add an invariant test config in acceptance/bundle/invariant/configs/<resource>.yml.tmpl
   - See existing configs in that directory for the format.
 - Add bind/unbind tests in acceptance/bundle/deployment/bind/<resource>/
   - These verify that binding an existing resource and then deploying/destroying works correctly.
 - For new resource types, run at least one test on a live cloud environment.
