# PyDABs resource acceptance tests

Each `<plural>-support/` directory is the acceptance test for one PyDABs resource. It
checks that the resource loads both from YAML and from Python and that a mutator runs
over it. `test_python_support_coverage`
(`python/databricks_tests/core/test_python_support.py`) requires every PyDABs resource
to have one, so a newly-onboarded resource needs a fixture here.

Copy an existing one — `alerts-support/` (a resource with required nested fields) or
`catalogs-support/` (direct-engine only) are the canonical examples. A fixture is six
files:

- `databricks.yml` — `bundle.name: my_project`, `sync: {paths: []}`, a top-level
  `python:` block wiring `resources:load_resources` + `mutators:update_<singular>`, and
  one YAML-declared instance `<plural>.my_<name>_1`.
- `resources.py` — `load_resources()` adds a second instance `my_<name>_2` via
  `resources.add_<singular>(...)`.
- `mutators.py` — a `@<singular>_mutator` that appends `" (updated)"` to a required
  string field; it runs over both instances.
- `script` — copy it verbatim (`bundle validate --output json | jq "pick(...)"`).
- `test.toml` — `Cloud = false`.
- `output.txt` — generated, never hand-written.

## Authoring a new one

1. Confirm the resource is wired: `python/databricks/bundles/<plural>/` exists, and
   `add_<singular>` / `<singular>_mutator` are in `databricks.bundles.core`. If not, it
   must be onboarded in PyDABs first.
2. Required fields are the `VariableOr[...]` (no default) fields in
   `python/databricks/bundles/<plural>/_models/<singular>.py`; set all of them,
   recursing into required nested objects. `VariableOrOptional[...] = None` fields are
   optional — omit them.
3. Get realistic values from `acceptance/bundle/invariant/configs/<singular>.yml.tmpl`,
   but **adapt**: replace `$UNIQUE_NAME` / `$TEST_DEFAULT_WAREHOUSE_ID` and other `$VAR`s
   with plain literals, and drop cloud-only blocks (`permissions`, `grants`,
   `file_path`) — this test is local and deterministic.
4. `test.toml`: add `EnvMatrix.PYDAB_VERSION = ["current"]` for a brand-new resource
   (it only exists in the current wheel), and `EnvMatrix.DATABRICKS_BUNDLE_ENGINE =
   ["direct"]` for a direct-only resource (terraform is deprecated — never a
   `["terraform", "direct"]` matrix). Match the newest fixture when unsure.
5. Generate the golden:
   `go test ./acceptance -run 'TestAccept/bundle/python/<plural>-support' -update`.
6. **Re-run without `-update`** — it must pass against the golden you just generated. A
   test that only passes with `-update` is nondeterministic (usually a `$VAR` or a
   volatile field left in); fix it before finishing.

Note: `bundle validate` normalizes the `python:` key to `experimental.python` in the
output — that's expected.
