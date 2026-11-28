The types in this package are equivalent to the lumberjack protos defined in Universe.
You can find all lumberjack protos for the Databricks CLI in the `proto/logs/frontend/databricks_cli` directory.

`databricks air run` logs one `air_run_event` per submission: requested compute,
which config options were set, the outcome and latency, and how long packaging and
uploading the code snapshot took. It never includes code, paths, names, parameters,
or image references. Set `DATABRICKS_CLI_DISABLE_TELEMETRY=1` to opt out.

The CLI logs an `aitools_hint_event` for each install-hint line it prints for a coding agent
(Claude Code's native `<claude-code-hint>` line and its text line are one event each; another
agent's text line is one event) and for the `apps init` skills tip or install prompt. The install
hint is never printed on hint-exempt commands (`hintExemptCommands` in
`libs/aitools/agents/hint.go`: aitools, `auth`, one-shot lookups, help, completion) or in CLI
processes the CLI started itself. It carries only the kind of hint (`hint_kind`); the command it was
shown on is the entry's `execution_context.command`. The event uploads at exit only when the command
resolved auth. The text line prints at most once per hour per agent session (or per agent on the
machine, for agents without a session id). Claude Code's native line prints at most once per 10
minutes per session, but its event is logged at most once per text-line print: only for the first
native line printed since the text line last printed. So when the command that printed a line
resolves no auth (e.g. `databricks bundle schema`), no event for that line is uploaded until the
text line prints again, even though the agent saw the hint.

`databricks feedback` sends one `fast_feedback_log` entry, the event the in-product feedback widget
records, instead of a `databricks_cli_log`. This command explicitly submits feedback, so it uploads
immediately in a single attempt even when `DATABRICKS_CLI_DISABLE_TELEMETRY` is set. AI agents can
submit directly without user confirmation. A successful submission prints a receipt. Agent and
run context ride in the entry's `user_agent`.

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

On the first run of a new CLI version, the CLI can install Databricks AI tools for the coding agents
on the machine (see `databricks aitools auto-install --help`). It logs an `aitools_auto_install_event`
with the outcome (`INSTALLED`, `UP_TO_DATE`, `KEPT`, `SKIPPED_CI`, `SKIPPED_OPTOUT`,
`SKIPPED_HOLDOUT`, `SKIPPED_NO_AGENTS`, `SKIPPED_PLATFORM`, `FAILED`, `RETRIES_EXHAUSTED`), the
machine's holdout arm and, for a failure, its error category: once per version and machine, plus
once per retry of a failed attempt. `DECLINED` and `UNANSWERED` are no longer sent: auto-install
no longer asks. Nothing is logged for a version whose outcome is settled, on Databricks Runtime, or
while the feature is turned off remotely (the CI and opt-out checks come first and are logged
either way). A run on a machine whose user has not agreed to auto-install logs nothing and leaves
the version to a later run. An install with no tools installed yet also logs an
`aitools_install_event` with `trigger` `AUTO_FIRST_RUN`; installs by `databricks aitools install`
leave `trigger` unset. The holdout arm comes from a random machine id kept in
`~/.databricks/aitools/auto-install.json`; the id itself is never sent. Like the other aitools
events, these upload at exit only when the command resolved auth.
