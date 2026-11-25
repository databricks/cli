# Bundle schema annotations

This directory holds the documentation the CLI owns for the bundle
configuration JSON schema: docs for fields that do not exist in the upstream
API spec (`.codegen/cli.json`), and overrides of upstream docs. Documentation
for everything else is inherited from `cli.json` at generation time and must
not be duplicated here.

## Files

The annotations form one tree that mirrors the bundle configuration, sliced
into files:

- `root.yml` holds the top-level keys (`artifacts`, `bundle`, `targets`, ...),
  including the `resources` node's own description.
- `<resource_type>.yml` (`jobs.yml`, `pipelines.yml`, ...) holds the node of
  that resource type under `resources`.
- `*.overrides.yml` (`jobs.overrides.yml`, ...) hold, sliced the same way,
  every hand-written entry for a field that `cli.json` already defines:
  description overrides, gap fills for undocumented upstream fields, and
  deprecations the spec does not carry. Each entry is a candidate to move
  upstream (fix the proto or OpenAPI docs) and then delete; the set of these
  files is the burn-down list. The generator routes entries here automatically,
  so you can add a node in any hand-written file and it lands in the right
  place on regenerate.
- `*.generated.yml` are written by `task generate-schema` and list the fields
  that have no documentation anywhere as `description: PLACEHOLDER`. Do not
  edit them. To document a field, add it to the matching hand-written file at
  the same tree position and regenerate; the placeholder then disappears.

Every entry in a hand-written file is authored. `task generate-schema` only
normalizes those files (key order, formatting) and drops entries for fields
that no longer exist in the configuration.

## Format

The `$type` and `$fields` keys are structural; every other key is a config
field name.

- A node documents one field: its inline keys (`description`,
  `markdown_description`, ...) apply to the field itself.
- `$type` documents the type the field's value resolves to; for map and
  sequence fields, each entry. These docs are shared by every occurrence of the
  type; enum values also live here.
- `$fields` holds the nodes of that type's fields (map and sequence levels are
  unwrapped implicitly).
- Each type is expanded exactly once, at its first occurrence in struct
  declaration order. Fields of types that occur again later are documented at
  that first occurrence. This is why types shared by several resources (for
  example `lifecycle` or `permissions`) are expanded in the file of the first
  resource declared in `config.Resources`, and why everything under `targets`
  is documented at the top-level keys it overrides.
