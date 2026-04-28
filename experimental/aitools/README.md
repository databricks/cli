# Experimental AI Tools

`databricks experimental aitools` is the remaining experimental surface for coding-agent workflows.

Current commands:

- `databricks experimental aitools skills list`
- `databricks experimental aitools skills install [skill-name]`
- `databricks experimental aitools install [skill-name]`
- `databricks experimental aitools tools query`
- `databricks experimental aitools tools discover-schema`
- `databricks experimental aitools tools get-default-warehouse`

Current behavior:

- `skills install` installs Databricks skills for detected coding agents.
- `install` is a compatibility alias for `skills install`.
- `tools` exposes a small set of AI-oriented workspace helpers.
- `tools query` accepts a single SQL or multiple SQLs in one invocation. Pass
  several positional arguments and/or repeat `--file` to run them in parallel
  against the warehouse. Multi-query output is always JSON; control parallelism
  with `--concurrency` (default 8).

  ```bash
  databricks experimental aitools tools query \
    --warehouse <wh> --output json \
    "SELECT count(*) FROM samples.nyctaxi.trips" \
    "SELECT min(tpep_pickup_datetime), max(tpep_pickup_datetime) FROM samples.nyctaxi.trips" \
    "SELECT vendor_id, count(*) FROM samples.nyctaxi.trips GROUP BY 1"
  ```

Removed behavior:

- there is no MCP server under `experimental aitools`
- the old `deploy` and `validate` flows were removed
- command names and behavior in this area are still experimental and may change
