# Example gallery

One test file per resource / capability the framework supports today. Each file's header
lists precisely what it CAN and CANNOT test locally. New files land here as the framework
grows, so this folder doubles as a record of capability over time.

Supported now (local DuckDB backend):
- `test_job_sql.py` — SQL job: run the real `.sql`, assert on output tables
- `test_job_config.py` — read a resource's declared wiring (no execution)
- `test_job_nonsql.py` — non-SQL job: skips loudly (boundary demo)
- `test_volume.py` — volume upload + read the file back (row count, columns)

Not yet (need the cloud backend or new handles):
- pipelines (Lakeflow/DLT) — run + assert on output tables
- dashboards — assert source tables / wiring
- alerts, permissions, clusters — config + live state
- running Python / Scala / R / notebook jobs
