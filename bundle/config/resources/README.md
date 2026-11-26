# Bundle resource types

This package holds one Go type per resource type that a bundle can declare under `resources:`.
This document defines where logic about a resource type lives, so that shared code never needs a list of resource types.

## Rules

1. Static facts about a type (permissions, grants, URL pattern, approval messages, ...) are declared in the type's own file in this directory.
2. Behaviour on a type's own typed config is a method on the type, reached through a small optional interface.
3. Behaviour that needs the whole bundle (mutators, validators, runners, generators, engine adapters, fake-server APIs) lives in one file per type (or per API) in the package that owns that behaviour, and registers itself from that file.
4. Data lives in one file per type in a directory that is loaded by glob.
5. `config.Resources` (`bundle/config/resources.go`) is the only hand-written list of resource types. Everything else is derived from it.

## Conventions

- **Registration:** a per-type file registers itself with its package's registry from `func init()`. Registration only records the value; any non-trivial work happens on first use. A registry must not depend on registration order: it sorts by resource key (or another explicit, documented order) before use. If a type registers more than once, the registry documents how those registrations are ordered, and that order must not depend on the order in which `init()` functions run.
- **File names:** a new per-type file is named after the type's key under `resources:` (`jobs`, `pipelines`, `feature_specs`): `<key>.go`, or `<key>_<aspect>.go` when a type has several files.
- **Completeness by test, not by list:** where a fact must be declared for every type, a test fails for a type that declares nothing and names that type.
- **Shared code names no resource type:** shared ("engine") code contains no resource-type names, no type switches on resource types and no per-type branches. Such logic goes where the rules above put it. `TestSharedCodeNamesNoNewResourceTypes` (`shared_code_test.go`) fails when a shared Go file under `bundle/` or `cmd/bundle/` names a resource type that `testdata/shared_code_resource_types.txt` does not list for that file; the file lists the shared code that still names a type, and shrinks as that code moves to per-type files.

## Adding a resource type

This is the list of files a new resource type needs, in the order you would write them. `<key>` is the type's key under `resources:` (`feature_specs`). Each step names the test that fails while the step is missing. The files of `feature_specs` are a complete example: `git ls-files '*feature_spec*'` lists them.

1. **The field.** In `bundle/config/resources.go`, add a field `map[string]*T` to `config.Resources`, with `<key>` as its JSON tag. Its position is the type's place in the canonical order. Nothing else in that file changes; it compiles once step 2 adds `T`.
2. **The config type.** `bundle/config/resources/<key>.go` holds the type `T`. It usually embeds `BaseResource` and the SDK struct of the API; a type that embeds an SDK struct implements `MarshalJSON` and `UnmarshalJSON` (`TestCustomMarshallerIsImplemented`). `*T` implements `config.ConfigResource` (`Exists`, `ResourceDescription`, `GetName`; `BaseResource` provides `GetLifecycle`). The type declares its facts as methods:

   | Method | Declared by | Test |
   |---|---|---|
   | `RunAsSupport()` | every type | `TestRunAsSupportIsDeclaredForEveryResourceType` |
   | `Approval()` | every type (`Approval{}` if deleting or recreating needs no warning) | `TestEveryResourceTypeDeclaresApproval` |
   | `WorkspaceURL()`, and `InitializeURL()` if the type has a page | every type | `TestEveryResourceTypeDeclaresWorkspaceURL`, `TestResourceTypesWithWorkspacePageInitializeURL` |
   | `PermissionsSupport()` | types with a `Permissions` field | `TestPermissionsDeclaredIffPermissionsField` |
   | `GrantsSupport()` | types with a `Grants` field | `TestGrantsDeclaredIffGrantsField` |
   | `UCNames()` | types with fields that name a catalog or schema | `TestResourceTypesWithUCNameFieldsDeclareThem` |
   | `ApplyPresets()` to prefix the name in development mode, or `NamePrefixExemptReason()` | every type | `TestAppropriateResourcesAreRenamed`, once step 13 registers examples of the type |
   | `LifecycleStartedTelemetryKey()` | types whose lifecycle supports `started` | `TestLifecycleStartedTelemetryIsDeclaredByTypesWithStarted` |

   The other optional interfaces of this package (local paths, library locations, field defaults, schema transforms, ...) are declared in its shared files, one concern per file; implement the ones that apply.
3. **The bind fixture.** `bundle/config/resources/<key>_bind_test.go` calls `registerBindFixture` from `init()`, or `registerNotBindable` with the reason. Test: `TestResourcesBindSupport`.
4. **Schema annotations.** `bundle/internal/schema/annotations/<key>.yml` documents the fields only the CLI defines. `./task generate-schema` writes the other files of the type there and `bundle/schema/jsonschema.json`; see the README in that directory. Test: `TestRequiredAnnotationsForNewFields`.
5. **The direct engine adapter.** `bundle/direct/dresources/<key>.go` implements the adapter and calls `registerAdapter("<key>", ...)` from `init()`. The adapters of the type's permissions and grants follow from step 2. Test: `TestEveryResourceTypeHasAdapter`. See `bundle/direct/dresources/README.md` for how to implement it.
6. **The direct engine test fixture.** `bundle/direct/dresources/<key>_test.go` registers the input config of the type and of its permissions and grants from `init()`. Tests: `TestEveryAdapterHasTestFixture`, `TestAll/<key>`, `TestCreateRecoveryCoverage/<key>`. The last requires the adapter's `DoCreate` to recover from a failed create request (`retrySafe` or `recoverCreate`), or the test file to call `registerCreateRecoveryOptOut` from `init()` with the reason; see "Failed create requests" in `bundle/direct/dresources/README.md`.
7. **The direct engine field rules.** `bundle/direct/dresources/configs/<key>.yml` holds the hand-written rules, if the type needs any; no unit test requires it, the acceptance tests of the type fail without the rules they depend on. `./task generate-direct` writes `configs/<key>.generated.yml` and the type's entry in `apitypes.generated.yml`. Test: `TestNoUpdateResourcesCoverAllFields/<key>`, for a type whose adapter has no `DoUpdate`.
8. **The fake server.** `libs/testserver/<api>.go` models the API: it calls `registerStore` and `registerRoutes` from `init()`, keyed by its file name. Tests: `TestAll/<key>`, `TestCreateRecoveryCoverage/<key>` and the acceptance tests.
9. **Acceptance tests.** `acceptance/bundle/resources/<key>/` and `acceptance/bundle/deployment/bind/<key>/`. Test: `TestResourceTestDirsCoverage` (`acceptance/resource_test_dirs_test.go`); a type that cannot have one is listed in `LackingResourceTests` or `LackingBindTests` there, with the reason.
10. **The invariant config.** `acceptance/bundle/invariant/configs/<key>.yml.tmpl`, with optional `-init.sh`, `-cleanup.sh` and `-exclude.toml` sidecars; see `acceptance/bundle/invariant/README.md`. Test: `TestInvariantConfigsCoverage`.
11. **Python support.** `./task pydabs-codegen` writes the Python models and wiring under `python/`. `acceptance/bundle/python/<key>-support/` tests them; see `acceptance/bundle/python/README.md`. Tests: `test_python_support_coverage`, and `test_core_public_api`, whose golden `python/databricks_tests/core/public_api.txt` is regenerated with `UPDATE_SNAPSHOTS=1`.
12. **The rest of the generated files.** `./task generate` also writes `acceptance/bundle/refschema/output/<key>.txt` (test: `TestAccept/bundle/refschema`) and `bundle/internal/validation/generated/`. Acceptance outputs that list every resource type (`acceptance/experimental/open/output.txt`, the invariant tests' `out.test.toml`) are regenerated with `-update`. The CI job that runs `./task generate-check` fails on any generated file that is not up to date.
13. **Target mode examples.** `bundle/config/mutator/resourcemutator/<key>_target_mode_test.go` calls `registerTargetModeExamples` from `init()` with example resources of the type, which the target mode tests run on. Test: `TestAllResourcesMocked`.
14. **The changelog entry.** `.nextchanges/bundles/<name>.md`; see `.nextchanges/README.md`.

Optional acceptance tests that no coverage test requires: grants (`acceptance/bundle/resources/grants/<key>/`) and implicit dependencies (`acceptance/bundle/resource_deps/`). Helpers for the type's acceptance tests (`acceptance/bin/`, `script.prepare`) are needed only by the tests that call them.

### Behaviour that needs the whole bundle

A type that needs more than its declarations adds one file per behaviour in the package that runs it, named `<key>.go` or `<key>_<aspect>.go`, and registers it from `init()`. Nothing checks that a type registers these; the tests of the behaviour do.

| Behaviour | File | Registration |
|---|---|---|
| Mutator or validator at a point of the pipeline | `bundle/config/mutator/resourcemutator/<key>_<aspect>.go`, `bundle/config/validate/<key>_<aspect>.go` | `stages.Register(stage, "<key>", ...)`; the stages are documented in `bundle/config/stages/stages.go` |
| `bundle run` | `bundle/run/<key>.go` | `registerRunner` |
| `bundle generate` | `cmd/bundle/generate/<key>.go` (helpers in `bundle/generate/<key>.go`) | `registerGenerator` |
| `bundle config-remote-sync` | `bundle/configsync/<key>_<aspect>.go`, rules in `bundle/configsync/rules/<key>.yml` | `registerChangeHook`, `registerResourceIDField` |
| Check before plan and deploy | a file in `bundle/deploy/` (`check_dashboards_modified_remotely.go`) | `preDeployChecks.Register` |
| Deployment metadata | `bundle/deploy/metadata/<key>_compute.go`, and for annotations a file in `bundle/deploy/metadata/` (`annotate_jobs.go`) | `resourceMetadata.Register`, `annotations.Register` |
| Check for running resources | `bundle/statemgmt/<key>_running.go` | `runningChecks.Register` |
| Destroy approval that depends on state | `bundle/phases/<key>_approval.go` | `registerDestroyApprovalHook` |
| Deploy telemetry | `bundle/metrics/<key>_<aspect>.go` | `DeployBoolValues.Register` |

## Finding what a type declares

The canonical order of resource types is the field order of `config.Resources`.

- `config.AllResourceTypes()` returns every resource type in canonical order: its key, its Go type and its `ResourceDescription`. `New()` returns a fresh zero value of the type as `config.ConfigResource`.
- `config.ResourceTypesImplementing[I]()` yields the key and a fresh zero value of every type whose zero value implements the interface `I`, in canonical order:

  ```go
  for key, d := range config.ResourceTypesImplementing[resources.SomeDeclarer]() {
      // d is a zero value of the type under key, as resources.SomeDeclarer.
  }
  ```

- `config.ResourceTypeAs[I](key)` returns a fresh zero value of the type under `key` as `I`, and false if there is no such type or it does not implement `I`.

A declaration must not depend on the receiver's contents: it is called on a zero value.

## Registering behaviour

A package that needs at most one value per type (a check, a contribution function, ...) keeps a `config.ResourceTypeRegistry[T]` in a package-level variable, and the file of each type calls its `Register` from `init()`. `All()` yields the registrations in canonical order, whatever the order of `init()`, and `Get(key)` returns one. On first use the registry panics on a key that is not a resource type or that is registered twice, and `Register` panics once the registry has been used.

`stages.Register` takes several registrations per type: the mutators of one call run in the given order, and the calls of a type for one stage run in the order of the name of each call's first mutator.

## Testing completeness

`config.AssertResourceTypesImplement[I](t, except...)` fails the test for every resource type that does not implement `I` and is not listed in `except`, and names its key. It also fails for a listed key that is not a resource type or that does implement `I`, so an opt-out cannot go stale. Give each listed key a comment with the reason:

```go
func TestEverySomeDeclarer(t *testing.T) {
	config.AssertResourceTypesImplement[resources.SomeDeclarer](t,
		// A job run has no ...
		"job_runs",
	)
}
```
