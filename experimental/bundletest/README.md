# bundletest

Experimental, pytest-style **isolation testing** for Databricks Asset Bundles (DABs).

## Quick start

Install the package in a bundle project, generate a starter test, and run locally:

```sh
uv pip install -e /path/to/cli/experimental/bundletest
databricks bundle test init
databricks bundle test --local
```

`bundletest init` finds the nearest `databricks.yml`, chooses a declared resource, and
creates `tests/conftest.py` plus a marked `tests/test_bundle.py`. It refuses to overwrite
either file unless `--force` is supplied.

Every local run begins with a support report, so unsupported tasks are visible before
pytest starts:

```text
bundletest local support: orders

[LOCAL ] jobs.transform_orders/transform: runs src/transform_orders.sql
[LOCAL ] jobs.aggregate_orders/aggregate: runs src/aggregate_orders.sql
[CLOUD ] jobs.score_model/score: notebook_task requires a Databricks workspace
[CONFIG] 33 non-job resources: configuration assertions only

summary: 2 local, 1 cloud-only, 33 config-only
```

Use `databricks bundle test --support-only` for the report without a test run.

Unit tests answer "does my function return the right value?" `bundletest` answers the
next question up: **"does my deployed bundle resource actually produce the right table?"**
That class of bug — a job wired to the wrong upstream, a renamed table, a transform
regression — passes every unit test and only shows up after `databricks bundle deploy` +
a real run.

## The idea: test one component in isolation

You write a small test for a *single* resource. You stand in for its upstream neighbors
with `env.seed(...)`, run that one resource's **real deployed SQL**, and assert on its
real output:

```python
def test_transform_dedupes(env):
    env.seed(
        "shop.bronze.raw_orders",
        [
            {"order_id": 1, "total_price": 10.0},
            {"order_id": 1, "total_price": 10.0},  # duplicate
            {"order_id": 2, "total_price": 5.0},
            {"order_id": None, "total_price": 1.0},  # null id -> filtered out
        ],
    )
    env.run_job("transform_orders")  # runs src/transform_orders.sql for real
    assert env.table("shop.silver.orders").row_count() == 2
    assert env.table("shop.silver.orders").has_no_nulls("order_id")
    assert env.table("shop.silver.orders").column("order_id").is_unique()
```

We isolate by substituting the component's **data-boundary neighbors**, never by faking
the component's own output — faking the thing under test is a tautology that catches
nothing.

Job runs are checked by default. A failed task raises an assertion with the resource,
task, source file, backend, run ID when available, and the original error:

```text
bundle job 'transform_orders' failed
  task: transform
  source: src/transform_orders.sql
  backend: local
  error: Catalog Error: Table raw_orders does not exist
  use check=False to inspect an expected failure
```

Expected-failure tests can opt out explicitly:

```python
result = env.run_job("transform_orders", check=False)
assert not result.succeeded
```

## Two backends, one seam

The same test runs against either backend, chosen by the `BUNDLETEST_BACKEND` env var:

- **`local`** (default) — **portable SQL smoke testing**: runs the job's *actual* deployed
  `.sql` artifact against [DuckDB](https://duckdb.org). Zero infra, seconds to run. Because
  it runs the same source of truth the bundle deploys (with strict typing, no silent
  coercion), it catches the structural bugs portable SQL can express — wrong table name,
  broken wiring, dedup/null-filter regressions. It is **not** a Databricks SQL emulator:
  DuckDB's dialect, type system, and semantics differ, so a green local run means "the SQL
  is portable and structurally sound," not "this passes on Databricks." Genuinely
  dialect-dependent checks belong on cloud.
- **`cloud`** — a real workspace. Real fidelity: `databricks bundle deploy` + real job runs
  and SQL through the Databricks SDK. Slower (deploys take minutes) and costs real compute,
  so it's the gated tier. The `cloud_only` assertions run here instead of skipping.

### How the local backend stays honest

- **Real artifact, not a reimplementation.** `env.run_job` reads the job's `sql_task`
  file from `databricks.yml` and runs that exact text.
- **Names bound at the environment level, query body never rewritten.** Seeded tables and
  target namespaces are created under their real `catalog.schema.table` names (DuckDB
  `ATTACH` / `CREATE SCHEMA`) so the unmodified SQL resolves against them.
- **Three-way routing, never a false green *or* a false red:**
  - a missing table/column is a real bug → the run **fails** (red);
  - a Databricks-only SQL function, a notebook/Python task, or a reserved catalog name
    (`main`/`temp`/`system`) can't be judged locally → `LocalUnsupported` → the test
    **skips with a reason**;
  - anything that runs and disagrees with an assertion → **red**.
- **Config resolved by the CLI's own engine, online references skipped loudly.** Deploy
  runs the real offline resolution (`cmd/offline-resolve`) — includes, target overrides,
  presets, and `${var.*}`/`${bundle.*}` all resolve exactly as `bundle validate` renders
  them, with no auth or network and no reimplementation. A reference only the workspace can
  resolve — `${workspace.*}`, a `lookup` variable, or an unset variable — is
  `LocalUnsupported` at the use site, never silently passed through.

Assertions you *know* are cloud-only (Databricks type naming, SLA timing, permissions)
can also be fenced explicitly with `@pytest.mark.cloud_only`, which skips them on any
non-cloud backend.

## Run it

The local backend resolves bundle config with the in-repo Go helper
`cmd/offline-resolve`, so a Go toolchain (matching the repo's `go.mod`) and a checkout of
the CLI repo are required in addition to Python.

```sh
uv venv --python 3.12
uv pip install -e ".[dev]"
databricks bundle test --bundle examples/orders_bundle --local -v
```

Arguments that `bundletest` does not consume are passed to pytest, so `-k`, `-x`, `-v`,
node IDs, and plugins continue to work normally. Use `--no-support-report` for compact CI
output.

### Run tests affected by a change

Mark each test with the resources it exercises:

```python
@pytest.mark.bundle_resource("jobs.transform_orders")
def test_transform_dedupes(env): ...
```

Then select tests from the Git diff:

```sh
databricks bundle test --local --changed --base origin/main
```

`bundletest` maps changed paths such as `src/transform_orders.sql` back to the bundle
resources that reference them. It runs tests with matching `bundle_resource` markers and
always includes changed test files. A YAML change runs the complete suite because it can
alter resource wiring, variables, or targets. The selection includes committed, staged,
unstaged, and untracked files.

### Run it on cloud

The cloud backend deploys to a real workspace. Example fixture `examples/cloud_orders/` contains two SQL jobs, a managed volume, and a file_path dashboard under `main.bundletest_cloud`:

```sh
export BUNDLE_VAR_warehouse_id=<sql-warehouse-id>
databricks bundle test --cloud \
  --bundle examples/cloud_orders \
  --profile <profile> \
  --warehouse-id <sql-warehouse-id>
```

The command requires `--profile` for cloud runs; it never selects a Databricks profile
implicitly. Add `--target <target>` when the bundle has a dedicated test target.

(`examples/orders_bundle/` is local static-config only, not deployable to cloud.)

Seeded tables and job runs are real and cost money, so unlike the local backend (a fresh
in-memory DuckDB per test) the cloud backend persists state within a run. `teardown()` drops
the tables it seeded and runs `bundle destroy`, but the tests still share a workspace — prefer
a **module-scoped** `env` fixture (deploy once per module) and an isolated namespace per run
over the local backend's function-scoped, throwaway one.
