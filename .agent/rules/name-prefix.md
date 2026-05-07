---
description: Rules for resource name prefixing in apply_presets
globs:
  - "bundle/config/mutator/resourcemutator/apply_presets.go"
  - "bundle/config/mutator/resourcemutator/apply_target_mode.go"
  - "bundle/config/mutator/resourcemutator/apply_target_mode_test.go"
  - "bundle/direct/dresources/*.go"
paths:
  - "bundle/config/mutator/resourcemutator/apply_presets.go"
  - "bundle/config/mutator/resourcemutator/apply_target_mode.go"
  - "bundle/config/mutator/resourcemutator/apply_target_mode_test.go"
  - "bundle/direct/dresources/*.go"
---

# Rules for resource name prefixing

`presets.name_prefix` and dev-mode renaming live in `bundle/config/mutator/resourcemutator/apply_presets.go`. They prepend a string to a resource field so deployments from the same bundle don't collide across users or targets.

**RULE: Only prefix display-name fields. Never prefix a field the API treats as the primary key / object id.** Prefixing an identity-bearing field changes which remote resource the bundle addresses, not just how it's labeled. The user sees a different name in the UI; the deployment also points at a different object.

To check whether a field is identity-bearing, look at the matching `bundle/direct/dresources/<resource>.go`:

- If `DoCreate` returns the name as the deployment id (e.g. `id := config.Name`) and `DoRead`/`DoUpdate`/`DoDelete` look the resource up by that name, the name is the primary key — do not prefix it.
- If the name is purely cosmetic and the API addresses the resource by a separate id (numeric, UUID, etc.), prefixing is fine.

When you decide a resource's name must not be prefixed, also add the resource type to `notRenamedFields` in `bundle/config/mutator/resourcemutator/apply_target_mode_test.go`. `TestAppropriateResourcesAreRenamed` enforces the invariant for every resource and will fail loudly if a future change reintroduces prefixing.
