# bundletest

Experimental, pytest-style **isolation testing** for Databricks Asset Bundles (DABs).

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
- **`cloud`** — deco-provisioned real workspace. Real fidelity. *(Arrives as a stacked PR
  on top of this base.)*

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

Assertions you *know* are cloud-only (Databricks type naming, SLA timing, permissions)
can also be fenced explicitly with `@pytest.mark.cloud_only`, which skips them on any
non-cloud backend.

## Run it

```sh
uv venv --python 3.12
uv pip install -e ".[dev]"
uv run pytest -v
```
