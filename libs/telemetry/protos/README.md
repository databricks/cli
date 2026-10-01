The types in this package are equivalent to the lumberjack protos defined in Universe.
You can find all lumberjack protos for the Databricks CLI in the `proto/logs/frontend/databricks_cli` directory.

AIR submission attempts use `databricks_cli_log.air_run_event`, mirrored by
`air_run.proto` in Universe. They record accelerator type/count, node count,
configuration-presence flags, retries, submission outcome and latency, Git usage,
and the successful Jobs run ID. `code_source_size_bytes` measures the compressed
tarball, using local file metadata for uploads and remote metadata for cache hits.
It is absent when no archive was produced or its size could not be measured.
Source contents, paths, dependency names, parameters, and image names are not sent.

`code_source_packaging_mode` distinguishes `GIT_ARCHIVE` from `PLAIN_TAR`, including
on cache hits. `code_source_packaging_duration_ms` measures archive creation only,
excluding source discovery and cache lookup. `code_source_upload_duration_ms`
measures the tarball write, including directory creation and retries. Both are
wall-clock milliseconds and remain available when the attempted phase fails.
Cache hits explicitly report zero for both durations; phases never attempted
because of an earlier failure, and runs without code sources, omit the duration.

The shared CLI logger uploads at command exit and respects
`DATABRICKS_CLI_DISABLE_TELEMETRY`. Dry runs do not emit a submission event.
Unlike Python AIR's `sgcli_log`, this event does not include `--via` attribution
or a separate first-log timing event, and watched submissions remain buffered
until the command exits. The Universe schema must land before ingestion can
retain the new event.
