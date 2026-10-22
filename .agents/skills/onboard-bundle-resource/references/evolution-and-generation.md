# API evolution and generation

Read this when extending an existing resource, auditing SDK coverage, or reducing onboarding boilerplate. The ordinary onboarding workflow describes current code. Design proposals below do not add capabilities to it.

## Extend or audit existing support

Start with the pinned Go SDK and `.codegen/cli.json`. In the SDK, `workspace_client.go` identifies services, `service/<package>/interface.go` exposes methods, `model.go` defines wire fields and serialization, and `impl.go` shows routes and request construction. Release notes identify leads; compare the actual old/new versions in scope. Resolve a user-supplied product name to its API entity and bundle plural before reporting a gap.

Classify findings using this table:

| Finding | Evidence required |
| --- | --- |
| Missing resource type | A deployable entity/lifecycle, API and bundle launch stage, identity and a missing config/adapter registration. A service is not necessarily one bundle resource; get/update-only singletons are valid candidates. |
| Newly supported update | An actual configuration update method plus a currently unsupported adapter/policy path. Data-plane methods such as vector-data upsert do not automatically belong in declarative deploy. |
| Missing field | A field absent from config/schema or dropped between input, state, request and readback. Lifecycle YAML lists special rules, not the complete field inventory. Check intentional output-only, preview and wrapper-only exclusions first. |
| Changed behavior | A concrete difference in names/IDs, defaults, masks, clearing, readback, asynchronous completion or deletion, established from the API contract and live evidence. |

For an audit, report the entity, pinned SDK/spec evidence, current CLI behavior, missing support or intentional exclusion, and the next focused change. Do not silently bump dependencies or implement all candidates. There is no checked-in `detect_sdk_drift.py` or resource-scaffold command to invoke.

For a selected implementation:

1. Preserve existing YAML/Python names, reference paths, bind identifiers and persisted state compatibility. A change to an already-shipped state shape may need `bundle/direct/dstate/migrate.go` handling.
2. Trace only the affected config, adapter, policy, mutator, fake and generated surfaces using the onboarding checklist. Confirm upstream annotations reach the correct generated API mapping before writing manual rules.
3. Use [API probing](../../probe-resource-api/SKILL.md) for new or changed semantics. Distinguish intended API behavior, observed behavior, and an adapter/marshaling bug. Read back after failed updates too: an error does not prove atomic rejection.
4. Test transition from the old configuration/state, the changed field, clearing/omission, and a subsequent no-drift plan. Preserve independent behavior assertions when replacing handwritten tables or helpers with generated data.

## Consolidated source material

The following efforts were compared on 2026-10-22. Their useful instructions are incorporated into the repo skills; their runtime implementations and package manifests remain separate.

| Source | What this workflow retains | Superseded assumptions |
| --- | --- | --- |
| [Plugin PR #10455](https://github.com/databricks-eng/plugin-marketplace/pull/10455), closed draft, head `5d375b823a07410cb63677224c7d13ab43fc4fe8` | Existing-resource evolution, SDK discovery, gap categories, destructive-action UX. Its `databricks-cli-add-dabs` / `databricks-cli-update-dabs` commands wrap instructions; it supplies no recording or generation tools. | Change-derived update masks, create-then-update as a default, nonmutating update mocks, automatic implementation from the latest SDK, monolithic lifecycle YAML and nonexistent drift scripts. |
| CLI branch `janniklasrose/agents/skill-add-new-resource-to-dabs` at `6759f90c6` | Package READMEs own long-lived contracts; skills sequence work. Stateful fake handlers, interaction tests, scoped `ForceSendFields` copying and deletion guardrails. | `.agent/` paths, Make targets, removed schema override files, Terraform gating, and missing README links. |
| CLI branch `specwork/dabs-auto-generation` at `0ef1a1d0e`, based on `~/_specs/pub/cli/dabs-auto-generation/` | Explicit mechanical/manual boundary, independent parity and completeness checks, API-evolution workflow, and staged automation below. | Treating the metadata prototype as a complete generator, manual Python registries, new Terraform mappings, or deriving policy correctness from the same metadata being tested. |

Keep adapter contracts in `bundle/direct/dresources/README.md`, testing conventions in the existing testing docs, and live experiments in the probing skill. A future plugin should locate the CLI checkout and dispatch to these repo skills so its guidance follows the code rather than maintaining another implementation copy.

## Generation roadmap and current boundary

Current generators handle CLI command stubs, reflected field inventories, bundle schema/docs annotations, required/enum validation, direct lifecycle field rules, and public Python models/wiring/docs. Config wrappers, direct CRUD, fake behavior and most resource policy wiring remain handwritten.

The auto-generation prototype extends `ResourceDescription` with metadata, adds a `resources.All` registry, and derives two consumers: bundle permission levels and permission fixups. It does **not** migrate direct/testserver permissions, grants or run-as behavior, and it does not generate wrappers, adapters, fake handlers or scaffolds. None of its runtime changes is included in this instructions branch.

Before advancing that prototype:

- Update it to the current resource set and preserve existing public names/titles and comments. Validate exactly-one registration, descriptor agreement, and completeness independently of the implementation.
- Correct and narrow metadata to the consumer being migrated. At the inspected revision, the snapshot direct-support flag is wrong, grants/run-as metadata is incomplete, and Features is missing. Direct ACL support, bundle-level permission mappings, ACL routing/IDs and owner fixups are distinct capabilities: cluster policies and secret scopes demonstrate why one boolean/list cannot represent all of them.
- Migrate one consumer with independent before/after behavior expectations. Comparing two calls to the same derived function does not prove parity. Retain tests for per-resource policy and real API behavior.
- Keep later milestones separate: remaining permission consumers; grants/run-as tables after policy decisions; config/registry scaffolding; simple adapter scaffolding; then scoped SDK gap reporting. Do not add unused future metadata or public helpers in anticipation of those milestones.

Any future adapter generator must preserve `PrepareState`/`DoRead`/`RemapState` semantics, force-send ownership, identity conversions, stable references and state compatibility. `TestInputSubset` and `TestRemoteSuperset` check path compatibility with explicit exceptions for bundle-only and input-only fields; synthesis must not disguise genuinely missing remote behavior. Static masks, wait hooks and one mutation per lifecycle operation still apply.

Generate mechanical structure only where inputs are reliable. Use existing `MapGet`, `MapList` and `MapDelete` helpers for fake implementations where they match observed semantics; defaults, normalization, partial failures, async transitions, hierarchical deletion and permissions still require evidence. HTTP 404 alone cannot prove a handler is missing: a correctly registered GET for an absent object also returns 404.
