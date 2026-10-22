# SDK field-copy fixer

Run `./task fix-sdk-field-copies` from the repository root after an SDK bump.
`bump-sdk-post` runs it first, before generation checks and acceptance-output
updates. The fixer completes reviewed field-copy literals in
`bundle/direct/dresources`; other exhaustruct failures still require review.

## What a verified copy means

The annotation supplies the semantic intent: newly added compatible fields
belong to the existing copy from this particular source. For example:

```go
request := sql.EditWarehouseRequest{ // sdk:copy-from config
    // Existing assignments, including exceptions, remain here.
}
```

Place the directive on the literal's opening-brace line. The source must be an
identifier or selector chain, such as `config` or `remote.DeltaSyncIndexSpec`.
Each nested literal needs its own annotation; the tool does not infer sources.

For each missing destination field, the tool requires:

- The same exported Go field name on an accessible, unambiguous source field.
- The same explicit, nonempty JSON field name, excluding `json:"-"` metadata.
- Identical Go types, without adding conversions or pointer dereferences.
- No source access through pointer embeddings that could introduce a nil panic.

Existing source pointers and selector chains must already be valid at the copy
site. This is a reviewed precondition: the vector-index spec mapping, for
example, is inside an existing nil guard. Type matching does not prove nonnilness.

Go 1.27 promoted destination keys are supported for partially populated value
embeddings. An assignment of the whole embedded value already carries its fields
and needs no promoted-key additions. Source lookup respects promotion and shadowing.

Existing expressions stay intact: explicit zeros and nils, ID overrides, renamed
fields, enum casts, conversions, filters, and metadata handling. The tool never
invents `ForceSendFields`, update masks, zero defaults, or resource field policies.
It reports missing fields it cannot copy at annotated sites and leaves them for
lint and manual review. Unannotated literals remain manual too. Compilation or
directive errors stop the run before writes; proposed contents are type-checked
again before writing.

These checks establish structural compatibility, not backend semantics. They do
not prove that an API accepts a field on update, permits clearing it, or treats
it as input rather than output. An annotation therefore requires a review of the
whole mapping, including how future fields should behave.

## Audit and initial opt-ins

The initial audit covered every registered direct resource and its population
paths. The 26 annotated literals below follow reviewed copy contracts. The
[resource guidelines](../../bundle/direct/dresources/README.md) give `RemapState`
a particularly clear contract: copy matching remote fields into the state subset;
API-specific reconstruction belongs in `DoRead`.

| Resource or family | Annotated copies | Other population paths and exclusions |
| --- | --- | --- |
| Clusters | Remap; create and edit requests | Whole embedded preparation; read combines libraries, lifecycle, and lifted policy defaults. Resize/start/install operations remain selective. |
| Cluster policies | Remap; edit request | Preparation normalizes shadowing definition fields. Create forwards the whole SDK value. |
| Instance pools | Remap; edit request | Preparation/create use the whole SDK value. Planner rules still govern fields requiring recreation. |
| Jobs | Create request | Preparation/remap/reset use whole settings; read assembles settings and metadata. Existing nil ACL and its filter remain explicit. |
| Job runs | None | Mixed run/override/parameter sources, synthetic result and lifecycle, token replacement, and deliberate drift exclusions. |
| Pipelines | Edit request | Read combines spec with response-level Parameters/RunAs; whole embedded state also carries local cascade behavior. |
| SQL warehouses | Remap; edit request | Whole embedded preparation/create; read derives lifecycle. Existing WarehouseType enum conversions remain explicit. These two sites cover the mapping shape in PR #7155. |
| Model serving endpoints | Auto-capture and served-entity input helpers | Aggregate config uses conversions and deprecated fields; update requires paired set/unset behavior. Tags use deltas. |
| Catalogs, schemas, volumes | Each remap | Whole create config; update selects supported fields, identity/rename behavior, and verified clearable fields. |
| External locations | Remap | Update deliberately zeros output-only Effective* fields even though matching config fields exist. Rename and unsupported owner/isolation behavior stay manual. |
| Registered models | Remap | Update deliberately omits Owner despite its matching config field; rename is separate. Remap retains output metadata for drift rules. |
| MLflow models | Remap | Read wraps SDK output plus numeric ID; update response combines SDK fields, explicit zeros, and carried-forward ModelId. |
| Experiments | Prepare; remap | Create forwards config; update renames Name to NewName. Workspace-prefix normalization is separate. |
| Quality monitors | Nested SDK literal in remap | State shadows TableName for serialization. Update deliberately blanks failure metadata despite matching config fields. |
| Features | None | Whole SDK feature body; update overrides ID and forces Description with a description-only mask. |
| Database instances, database catalogs, synced database tables | None | Whole SDK objects inside request envelopes. Instance update separately overrides UID; deletion flags and waiters are operation-specific. |
| Postgres projects, branches, endpoints, databases, roles, catalogs, synced tables | None | Whole embedded specs already carry additions. Envelopes and bodies have different ID/parent responsibilities; optional specs, derived IDs, lifecycle flags, and masks are explicit. |
| Postgres snapshot schedules | None | Local state; derived branch/name; one update API handles create/update/delete using a fixed schedule-only mask. |
| Apps | None | Whole embedded App plus local config/lifecycle. Request builder clears SourceCodePath/GitSource because deployment uses a separate API; update has an explicit mask. |
| Alerts | None | Whole AlertV2 body inside request wrappers with explicit identity, mask, and purge behavior. |
| Dashboards | None | Draft/published sources, normalized parent, content conversion, publication state, selected request fields, and separate dataset selectors. |
| Genie spaces | None | Local config and converted content; update deliberately omits matching Etag to disable that concurrency precondition. |
| Secrets | None | PrepareState selectively copies the same SDK type while clearing output/effective fields; read normalizes effective values. Same name/type alone would copy unwanted output fields. |
| Secret scopes | Remap | Prepare/read rename scope/backend fields. Request-only InitialManagePrincipal remains explicitly empty on remap. |
| Secret-scope ACLs | None | Principal selection, permission conversion, parent references, and desired/current reconciliation. |
| Grants | None | Parsed identity, pagination, normalization, and privilege additions/removals assembled from desired and remote state. |
| Permissions | None | Typed/reflected input, parent references, inherited filtering, multiple sources, and Level-to-PermissionLevel rename. |
| Vector search endpoints | Remap | Remote ID/QPS reconstruction; update uses separate policy and target-QPS APIs. Request-only UsagePolicyId stays empty on remap. |
| Vector search indexes | Remap; guarded delta-sync spec conversion | Whole create config; separately resolved endpoint UUID and response/request spec conversion. The nested source remains behind its nil guard. |
| MCP services, model provider services, model services | None | Derived parent/ID and deliberately limited Comment/Config bodies; explicit output zeros and omitted etag preconditions. |
| Internal immutable snapshots | None | Entirely local state, derived paths, local-only fields, and upload results; no SDK subset requiring completion. |
| Alert-file loader outside direct resources | None | Also covered by exhaustruct, but YAML and file supply different authoritative fields; line-to-text transformations and source policy stay manual. |

The exclusions are deliberate. Source assignment counts, matching struct types,
or a large number of existing same-name copies cannot establish a copy contract:
external locations, secrets, apps, and Genie spaces all contain same-name fields
that their write paths intentionally omit. Existing masks and resource YAML also
remain independent requirements when adding support for a new SDK field.
