# `databricks query`

The `databricks query` command submits SQL statements to a Databricks SQL warehouse
through the [Statement Execution API](https://docs.databricks.com/api/workspace/sql-statements).
Use it for automation-friendly workflows such as scripts, CI jobs, or tooling that
requires structured responses.

## Running queries

Execute an inline statement:

```bash
databricks query sql \
  --warehouse-id <WAREHOUSE_ID> \
  --sql "SELECT 1 AS one"
```

Read the statement from a file (multiple statements separated by semicolons are
executed sequentially):

```bash
databricks query sql \
  --warehouse-id <WAREHOUSE_ID> \
  --file queries/sample.sql
```

## Output formats

The CLI renders results as a table when writing to an interactive terminal. You
can choose a specific format with `--format` and optionally redirect JSON or CSV
output to a file with `--result-file`.

- `table`: human-readable tabular output (default for terminals)
- `json`: newline-delimited JSON (NDJSON) with one object per row. Column names
  become JSON keys so the output can be loaded directly into Spark, pandas, or Polars.
- `csv`: comma-separated values. When multiple statements are provided, each block
  is separated by a blank line and repeats the header row.

Examples:

```bash
# NDJSON to stdout (default for non-interactive runs)
databricks query sql --warehouse-id <WAREHOUSE_ID> --sql "SELECT * FROM system.builtin.current_queries"

# NDJSON to a file
databricks query sql --warehouse-id <WAREHOUSE_ID> \
  --sql "SELECT * FROM system.builtin.current_queries" \
  --format json \
  --result-file current_queries.json

# CSV to a file
databricks query sql --warehouse-id <WAREHOUSE_ID> \
  --sql "SELECT id, email FROM users" \
  --format csv \
  --result-file users.csv
```

## Wait behaviour

The command waits up to 10 seconds for each statement to complete. After the
timeout it transparently polls until the statement reaches a terminal state.
Adjust the limit with `--wait-timeout`. Set it to `0s` to return immediately and
poll externally using the printed statement ID via
`GET /api/2.0/sql/statements/{statement_id}`. Addressing in-CLI polling is the
subject of follow-up work.

Valid values are `0s` or any whole-second duration between 5 seconds and 50
seconds, as required by the Statement Execution API.

## Read-only safety gate

By default the CLI blocks statements that may modify data or change workspace
state. The classifier removes comments, splits multi-statement files, and checks
the leading keyword of each statement. Read-only statements such as `SELECT`,
`SHOW`, `DESCRIBE`, `VALUES`, `TABLE`, `WITH … SELECT`, and `EXPLAIN SELECT`
are allowed. Statements that begin with keywords such as `ALTER`, `CREATE`,
`DELETE`, `DROP`, `INSERT`, `MERGE`, `SET`, `TRUNCATE`, `UPDATE`, `USE`, and
similar commands are rejected.

When a statement is blocked the error identifies the keyword, statement index,
and location. Override the protection when you intentionally run commands with
side effects:

- `--allow-destructive`
- `export DATABRICKS_CLI_ALLOW_DESTRUCTIVE_SQL=true`
- `cli.allow_destructive_sql = true` in the active profile inside `~/.databrickscfg`

Flag, environment variable, and profile are applied in that order of precedence.

## Large result sets

For large result sets Databricks may return chunk metadata or signed links to
external storage. The CLI streams inline chunks directly and downloads external
links sequentially, converting them into NDJSON or CSV on the fly. If a result
is truncated server-side (due to a row or byte limit), the CLI prints a warning
to stderr after the statement finishes.

## Arrow and other formats

The CLI currently requests JSON output (`JSON_ARRAY` format). Arrow output
(`ARROW_STREAM`) is intentionally out of scope for this release. If a profile or
workspace forces Arrow, the command fails with a clear error instructing you to
rerun with JSON output.
