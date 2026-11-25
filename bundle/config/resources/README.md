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
- **Shared code names no resource type:** shared ("engine") code contains no resource-type names, no type switches on resource types and no per-type branches. Such logic goes where the rules above put it.

## Adding a resource type

Add a field `map[string]*T` to `config.Resources`, with the key under `resources:` as its JSON tag, and make `*T` implement `config.ConfigResource`. Nothing else in `bundle/config/resources.go` changes. The completeness tests then name every fact the new type still has to declare.

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
