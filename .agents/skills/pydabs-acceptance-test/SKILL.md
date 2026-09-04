---
name: pydabs-acceptance-test
description: "Author the acceptance test for a PyDABs (Python DABs) resource. Use when onboarding a new PyDABs resource, when the test_python_support_coverage guard fails for a resource, or when the user says 'add a pydabs acceptance test', 'write the acceptance test for <resource>', or 'the <resource>-support test is missing'. Produces the acceptance/bundle/python/<plural>-support/ fixture (databricks.yml + resources.py + mutators.py + script + test.toml + generated output.txt)."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, AskUserQuestion
---

# Author a PyDABs resource acceptance test

PyDABs acceptance tests are hand-written, one fixture per resource under
`acceptance/bundle/python/<plural>-support/`. The OpenAPI spec has no examples, so
the realistic field values come from the resource's invariant config and your own
judgement — not from a generator. This skill guides you through authoring that
fixture deterministically and verifying it.

The coverage guard `test_python_support_coverage`
(`python/databricks_tests/core/test_python_support.py`) fails CI for any PyDABs
resource that lacks a `<plural>-support` fixture, so every newly-onboarded resource
must get one. This skill is how you close that gap.

Worked reference: `acceptance/bundle/python/alerts-support/` (a resource with required
nested fields) and `acceptance/bundle/python/catalogs-support/` (a direct-engine-only
resource). Read both before starting — the fixture you write mirrors them.

## Input

The resource to cover, as its **plural** name (the `resources:` key in
`databricks.yml`, e.g. `alerts`, `volumes`). If given a singular or a Go/Python
type name, resolve it to the plural first (step 1).

## Step 1 — Verify the resource is wired in PyDABs

The fixture cannot work unless the resource's Python surface exists. Confirm all of:

- The package `python/databricks/bundles/<plural>/` exists and has a `_models/`
  subdirectory (this is what marks it a generated resource package).
- `add_<singular>` is a method on `Resources` and `<singular>_mutator` is exported
  from `databricks.bundles.core`:

  ```sh
  grep -rn "def add_<singular>\|<singular>_mutator" python/databricks/bundles/core/
  ```

If any is missing, the resource is not wired yet — stop and onboard it in PyDABs
first (that is a separate task). Note the exact `<singular>` and `<CLASS>` names
(the dataclass, e.g. `Alert`) from `python/databricks/bundles/<plural>/__init__.py`;
you need them for `resources.py` and `mutators.py`.

## Step 2 — Find the resource's required fields

The generated dataclass is the source of truth. In
`python/databricks/bundles/<plural>/_models/<singular>.py`, required fields are typed
`VariableOr[...]` (no default); optional fields are `VariableOrOptional[...] = None`.
You must set every required field, including required fields of required nested
objects (recurse into their `_models` files). Optional fields are usually omitted.

```sh
grep -nE "VariableOr\[|VariableOrOptional\[" python/databricks/bundles/<plural>/_models/<singular>.py
```

## Step 3 — Get realistic values (adapt, don't copy)

The invariant config `acceptance/bundle/invariant/configs/<singular>.yml.tmpl` shows
realistic values for the same resource. **Adapt** it — do not copy verbatim:

- Replace `$UNIQUE_NAME`, `$TEST_DEFAULT_WAREHOUSE_ID`, and every other `$VAR`
  interpolation with plain string literals. This test runs locally with no cloud and
  no variable substitution.
- Drop cloud-only / IAM blocks the invariant config carries for its real-workspace
  run: `permissions`, `grants`, `file_path` pointing at an external asset, etc. Keep
  the fixture to the resource's own fields so `bundle validate` is deterministic.

If no invariant config exists, invent plausible literals that satisfy the field types
(a display name string, an enum's first member, a cron string, etc.).

## Step 4 — Write the six fixture files

Copy each `templates/*.tmpl` into `acceptance/bundle/python/<plural>-support/`,
dropping the `.tmpl` suffix (e.g. `resources.py.tmpl` → `resources.py`), and fill
them in. The `.tmpl` suffix keeps the placeholder files out of the repo's linters;
the copied fixture files are real Python/YAML/TOML and must pass lint (step 7).
Placeholders: `PLURAL`, `SINGULAR`, `CLASS`, `NAME` (a short stem, e.g. `alert`),
`FIELD` (a required **string** field to mutate).

1. **`databricks.yml`** — `bundle.name: my_project`, `sync: {paths: []}`, a top-level
   `python:` block wiring `resources:load_resources` and `mutators:update_<singular>`,
   and one YAML-declared resource `resources.<plural>.my_<NAME>_1` with all required
   fields. (`bundle validate` normalizes the `python:` key to `experimental.python`
   in the output — that is expected, don't fight it.)
2. **`resources.py`** — `load_resources()` adds a second instance `my_<NAME>_2` via
   `resources.add_<singular>(...)`, same required fields, slightly different values.
3. **`mutators.py`** — a `@<singular>_mutator` that `assert isinstance(...)` on a
   required string field and `replace(...)`s it to append `" (updated)"`. The mutator
   runs on **both** instances, so the golden shows the transform applied to each.
4. **`script`** — copy `templates/script` verbatim (`bundle validate --output json`
   piped through `jq "pick(.experimental.python, .resources)"`).
5. **`test.toml`** — `Cloud = false`. Add `EnvMatrix.PYDAB_VERSION = ["current"]` for
   a brand-new resource (it only exists in the current wheel, not the pinned older
   one). Add `EnvMatrix.DATABRICKS_BUNDLE_ENGINE = ["direct"]` only for a direct-only
   resource — terraform is deprecated, so never add a `["terraform", "direct"]`
   matrix. When unsure, copy the engine convention from the newest existing fixture
   (`catalogs-support`), not an old one.
6. **`output.txt`** — do NOT hand-write; generate it in step 5.

## Step 5 — Generate the golden output

```sh
go test ./acceptance -run 'TestAccept/bundle/python/<plural>-support' -update
```

(or `./task test-update` to regenerate the whole suite). This writes `output.txt`.
Inspect it: both `my_<NAME>_1` and `my_<NAME>_2` must appear with the mutated field
showing `" (updated)"`, and only `experimental.python` + `resources` keys are present.

## Step 6 — Verify it reproduces deterministically

Re-run **without** `-update`. It must pass against the golden you just generated:

```sh
go test ./acceptance -run 'TestAccept/bundle/python/<plural>-support' -count=1
```

A test that only passes with `-update` is nondeterministic — investigate before
finishing (a `$VAR` left in a value, a non-deterministic field, an EnvMatrix variant
producing different output). Never stop at "golden written".

## Step 7 — Confirm coverage and format

```sh
(cd python && uv run --python 3.11 pytest databricks_tests/core/test_python_support.py)
./task fmt && ./task lint-q
```

`test_python_support_coverage` should now be green for this resource. If the resource
was previously in the `_LACKING` allowlist
(`python/databricks_tests/core/test_python_support.py`), remove its entry — the list
only shrinks.
