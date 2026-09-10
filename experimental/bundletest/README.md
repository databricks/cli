# bundletest

Experimental, pytest-style **isolation testing** for Databricks Asset Bundles (DABs).

Unit tests answer "does my function return the right value?" `bundletest` answers the
next question up: **"does my deployed bundle resource actually produce the right table?"**
That class of bug — a job wired to the wrong upstream, a renamed table, a transform
regression — passes every unit test and only shows up after `databricks bundle deploy` +
a real run.

## The idea: test one component in isolation

You write a small test for a *single* resource. You stand in for its upstream neighbors
with `env.seed(...)`, run that one resource **for real**, and assert on its real output:

```python
def test_transform_dedupes(env):
    env.seed("bronze.raw_orders", [
        {"order_id": 1, "total_price": 10.0},
        {"order_id": 1, "total_price": 10.0},   # duplicate
        {"order_id": 2, "total_price": 5.0},
        {"order_id": None, "total_price": 1.0},  # null id -> filtered out
    ])
    env.run_job("transform_orders")             # the real transform runs
    assert env.table("silver.orders").row_count() == 2
    assert env.table("silver.orders").has_no_nulls("order_id")
    assert env.table("silver.orders").column("order_id").is_unique()
```

We isolate by substituting the component's **data-boundary neighbors**, never by faking
the component's own output — faking the thing under test is a tautology that catches
nothing.

## Two backends, one seam

The same test runs against either backend, chosen by the `BUNDLETEST_BACKEND` env var:

- **`memory`** (default) — sqlite-backed, runs today with zero infra. Fast test-shaping
  and CI smoke checks. Not high-fidelity (SQL dialect differs from Databricks SQL).
- **`cloud`** — deco-provisioned real workspace. Real fidelity. *(Arrives as a stacked PR
  on top of this base.)*

Assertions that depend on cloud-only behavior (SQL dialect/schema, SLA timing,
permissions) are fenced with `@pytest.mark.cloud_only` and **skip loudly** on the memory
backend — so a green local run never implies false confidence.

## Run it

```sh
uv venv --python 3.12
uv pip install -e ".[dev]"
uv run pytest -v
```
