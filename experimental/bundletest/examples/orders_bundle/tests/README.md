# Example gallery

One test file per resource / capability the framework supports. Each file's header lists
precisely what it CAN and CANNOT test locally. New files land here as the framework grows,
so this folder doubles as a record of capability over time.

## Execution (local DuckDB backend)
- `test_job_sql.py` — SQL job: run the real `.sql`, assert on output tables
- `test_job_nonsql.py` — non-SQL job: skips loudly (boundary demo)
- `test_job_config.py` — read a job's declared wiring (no execution)
- `test_volume.py` — volume upload + read the file back (row count, columns)
- `test_pipeline_end_to_end.py` — chain two SQL jobs (bronze → silver → gold), assert final table

## Resource handles — static wiring, read from `databricks.yml` (no workspace)

Every resource kind is reachable via the generic `env.resource(kind, name)` handle
(`.exists()`, `.config`, `.permissions()`, `.grants()`). Kinds that reference other
tables / artifacts / resources get a typed handle with a special accessor on top:

| Resource | typed handle | special accessor | example |
|---|---|---|---|
| pipelines | `env.pipeline` | `.catalog` / `.schema` / `.libraries()` | `test_pipeline.py` |
| dashboards | `env.dashboard` | `.source_tables()` | `test_dashboard.py` |
| genie_spaces | `env.genie_space` | `.source_tables()` | `test_genie_space.py` |
| quality_monitors | `env.quality_monitor` | `.monitored_table()` | `test_quality_monitor.py` |
| vector_search_indexes | `env.vector_search_index` | `.source_table()` / `.endpoint_name` | `test_vector_search_index.py` |
| model_serving_endpoints | `env.model_serving_endpoint` | `.served_models()` | `test_model_serving_endpoint.py` |
| apps | `env.app` | `.command()` / `.source_code_path` | `test_app.py` |
| jobs | `env.jobs[...]` | `.run()` / `.last_run()` | `test_job_config.py` |
| volumes | `env.volume` | `.upload()` / `.file()` | `test_volume.py` |

Every remaining user-authored kind (models, experiments, registered_models, catalogs,
schemas, external_locations, clusters, instance_pools, secret_scopes, secrets,
cluster_policies, sql_warehouses, alerts, vector_search_endpoints, database_instances,
database_catalogs, synced_database_tables, and the `postgres_*` family) is covered through
the generic handle — one example each in `test_config_resources.py`. `test_resource_generic.py`
shows the generic handle and the KeyError-safe `.exists()` for an undeclared resource.

## Not yet — needs the cloud backend
- running pipelines / dashboards / model endpoints and asserting on their live output
- running Python / Scala / R / notebook jobs
- server-defaulted or normalized config, and any live/deployed state
- introspecting a dashboard / genie space defined only by `file_path` (no inline queries) →
  skips loudly with `LocalUnsupported`
