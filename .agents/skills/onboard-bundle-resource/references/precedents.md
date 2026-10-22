# Examples and review lessons

Use the current code at these paths, then consult the linked reviews for the reason behind a decision. Earlier commits and PR descriptions can describe behavior that changed during review. These precedents were checked against main and PR history on 2026-10-22.

## Choose an example by API shape

| Shape | Example | Limits of the example |
| --- | --- | --- |
| SDK entity can serve as desired/read state | `bundle/config/resources/feature.go`, `bundle/direct/dresources/feature.go`, `configs/features*.yml` | Features permits only a `description` update mask. Its public grants API and DMS support were unavailable at onboarding. Those are resource-specific limitations. |
| AIP create has parent/leaf ID outside body | `bundle/config/resources/model_service.go`, `bundle/direct/dresources/model_service.go`, `configs/model_services*.yml` | JSON-visible config identity is reconstructed from the API name on read; mutable body uses `*`. Its bare UC deployment ID and intentional omission of optional etags are contract decisions, not universal conventions. |
| Same AIP family with different body fields | `bundle/config/resources/mcp_service.go`, `bundle/direct/dresources/mcp_service.go`, and the corresponding model-provider service files | Reuse the shape, but inspect each service's public grants support, defaults and nested fields. |
| Singleton with get/update only | `bundle/config/resources/postgres_snapshot_schedule.go`, `bundle/direct/dresources/postgres_snapshot_schedule.go` | Create is an update; delete disables the schedule. Inspect generated API mapping and actual launch stage. This resource is not a general CRUD or Python template. |
| Stable parent identity referenced by children | `configs/postgres_projects.yml`, `acceptance/bundle/resources/postgres_projects/update_keeps_children/` | The stable-output rule is useful; older Postgres change-derived masks and inline wait implementations should not be copied. |

Paths abbreviated as `configs/` above live under `bundle/direct/dresources/`. For the full adapter method/field contract, use its [README](../../../../bundle/direct/dresources/README.md).

## Reviews that changed the implementation

### Features: execute the behavior and activate the tests

[PR #6952](https://github.com/databricks/cli/pull/6952) initially added an invariant template without selecting it. [Review](https://github.com/databricks/cli/pull/6952#discussion_r4256438952) required the `invariant/test.toml` entry. Another [review](https://github.com/databricks/cli/pull/6952#discussion_r4256523914) changed immutable-field coverage from planning to actually deploying the recreate. [The revision](https://github.com/databricks/cli/pull/6952/commits/3d4b75fef93eeab24da6ef468dd22c3e4c272e1f) made both changes.

[Cloud review](https://github.com/databricks/cli/pull/6952#discussion_r4281301759) challenged whether a fake-only test demonstrated working support. Later commits enabled cloud lifecycle coverage; the original PR description's skipped-cloud statement became stale. The [final DMS exclusion](https://github.com/databricks/cli/pull/6952/commits/ed6b836391cbfb1b1ca3a7f5ab2270082d317d78) remained because the service did not accept the `features.` prefix. [#7039](https://github.com/databricks/cli/pull/7039) made the fake enforce the real service's prefixes.

[Grants review](https://github.com/databricks/cli/pull/6952#discussion_r4282072784) asked to include grants and exercise them in the invariant fixture. The [response](https://github.com/databricks/cli/pull/6952#discussion_r4291347849) established that an internal securable did not yet have a public grants API. Investigate this surface; do not manufacture unsupported config fields.

### AI Gateway: fix generation at the source and finish surrounding behavior

[PR #6525](https://github.com/databricks/cli/pull/6525) prompted reviews for a [working UI URL](https://github.com/databricks/cli/pull/6525#discussion_r3934566993), [UC dependency capture](https://github.com/databricks/cli/pull/6525#discussion_r3988652424), and [documented etag behavior](https://github.com/databricks/cli/pull/6525#discussion_r3981651920). These belong in the onboarding decision set even when CRUD is straightforward.

Initial manual output-only rules were challenged in [review](https://github.com/databricks/cli/pull/6525#discussion_r3978915150). The annotations already existed upstream: [#6618](https://github.com/databricks/cli/pull/6618) fixed traversal of reused nested schemas, and [#6654](https://github.com/databricks/cli/pull/6654) pinned `model_services: catalog.ModelService` in `apitypes.yml`, replacing manual rules with generated ones. A successful generator run does not prove it selected the right type or visited every path.

The [MCP service review](https://github.com/databricks/cli/pull/6633#discussion_r4014180155) explains why API response snapshots should project asserted fields: unrelated backend additions must not change the golden. Read the current tests alongside [#6633](https://github.com/databricks/cli/pull/6633), [#6634](https://github.com/databricks/cli/pull/6634), and the shared [grants follow-up #6635](https://github.com/databricks/cli/pull/6635).

### Singleton: reset semantics and generated launch stage

[PR #6449](https://github.com/databricks/cli/pull/6449) onboarded snapshot schedules. [Review](https://github.com/databricks/cli/pull/6449#discussion_r3902197881) corrected the assumption that delete was a no-op: it resets/disables the singleton, including when removed from bundle config. URL support changed repeatedly during review; the final implementation has no dedicated page. A [name review](https://github.com/databricks/cli/pull/6449#discussion_r3958140975) still required a meaningful canonical name.

Its initial Python exposure was later removed by [#6887](https://github.com/databricks/cli/pull/6887). The handwritten config type bypassed a launch-stage override because schema discovery returned early. Fixing generation applied the intended Beta stage and removed ineligible Python output. Check the generated schema's actual stage and Python eligibility; merely configuring an override is insufficient.

### Follow-up bugs: clearing and dependent recreation

[#6767](https://github.com/databricks/cli/pull/6767) tested Postgres update semantics against a real service: `*` rejected missing expected fields, while `spec` preserved omitted fields instead of clearing them. Thus a request accepted by the fake, or an update that changes a nonempty value, does not prove declarative replacement. Check removal, explicit zeros, and convergence separately. The new-resource convention remains static API masks or supported `*`, as required by the adapter README.

[#6865](https://github.com/databricks/cli/pull/6865) fixed parent updates delaying an unchanged output-only name and recreating children whose immutable parent referenced it. It introduced `stable_output_fields`, a guard test, and dependent-resource coverage. Stability depends on the API contract; membership in the state struct alone cannot establish it.

## WIP is evidence of open questions

[Data-quality-monitor PR #7094](https://github.com/databricks/cli/pull/7094) was open when this guidance was written. Reviews ask for [upstream field annotations](https://github.com/databricks/cli/pull/7094#discussion_r4302277501), [confirmation of wildcard update support](https://github.com/databricks/cli/pull/7094#discussion_r4334113429), and [acceptance coverage for presets](https://github.com/databricks/cli/pull/7094#discussion_r4334260686). The proposal had not validated its cloud lifecycle. Use these as questions to resolve, not proof that its custom mask, field suppression or no-URL choice is correct.
