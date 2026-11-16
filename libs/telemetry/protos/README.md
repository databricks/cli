The types in this package are equivalent to the lumberjack protos defined in Universe.
You can find all lumberjack protos for the Databricks CLI in the `proto/logs/frontend/databricks_cli` directory.

`databricks air run` logs one `air_run_event` per submission: requested compute,
which config options were set, the outcome and latency, and how long packaging and
uploading the code snapshot took. It never includes code, paths, names, parameters,
or image references. Set `DATABRICKS_CLI_DISABLE_TELEMETRY=1` to opt out.

The CLI logs an `aitools_hint_event` for each Claude Code install-hint line it prints (the native
`<claude-code-hint>` line and the text line are one event each) and for the `apps init` skills tip
or install prompt. The install hint is never printed on aitools commands, `auth token`,
`auth docker token` or in CLI processes the CLI started itself. It carries only the kind of hint
(`hint_kind`); the command it was shown on is the entry's `execution_context.command`. The event
uploads at exit only when the command resolved auth. The install hint prints at most once per
Claude Code session per hour, so when the command that printed it resolves no auth (e.g.
`databricks version`), that session logs no install hint event for the next hour even though the
agent saw the hint.

`databricks aitools` install, update, list, version and uninstall log an `aitools_command_event`
funnel per run: STARTED, one STEP_COMPLETED per install step reached (install only), and FINISHED
with the outcome and error category. The CLI sets no calling-agent field on it: the upload's
User-Agent carries the agent (`agent/<name>`), its Databricks tooling (`aitools-agent/...`) and the
`cmd-exec-id` that every event of the run shares. It uploads at exit only when the command resolved
auth. `list` and `version` upload only if auth has already resolved when they finish; they never
wait for it. Neither does a run of `install`, `update` or `uninstall` that ends before its plan or
confirmation step (a flag error, a cancelled prompt, no agent detected, a `--path` dump), nor an
`update` or `uninstall` that finds nothing installed. A run that does not wait still uploads,
within the upload timeout, when auth has already resolved.
