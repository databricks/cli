# AGENTS.md

Guidance for AI coding agents working in the **my_dbt_sql** project. This
is a dbt project deployed with a Declarative Automation Bundle (DAB): the dbt
models run as a Databricks job defined as code in `databricks.yml` and
`resources/`.

This file follows the cross-tool AGENTS.md convention and is also loaded by
Claude Code via CLAUDE.md.

## Project layout

- `databricks.yml` — the bundle definition: targets, variables, and `include`s.
- `resources/` — one file per resource (the job that runs dbt).
- `dbt_project.yml` — the dbt project configuration.
- `src/` — dbt models and other dbt sources.

## Working with the bundle

Use the Databricks CLI (the `databricks` skill has the current guidance):

- `databricks bundle validate` — type-check the configuration. Run this after
  every edit to `databricks.yml` or anything under `resources/`, and report the
  exact CLI error if it fails.
- `databricks bundle deploy --target dev` — deploy a development copy. `dev` is
  the default target and deploys in development mode (paused schedules, resources
  prefixed with your username).
- `databricks bundle run <resource_key>` — run the job that executes the dbt models.
- `databricks bundle summary` — inspect the resolved configuration.

Use `dbt` directly (e.g. `dbt run`, `dbt test`) for local model development; see
the project README for connection setup.

## Conventions

- Define each new resource as its own file under `resources/`; don't inline
  resources into `databricks.yml`.
- Never deploy to the `prod` target unless explicitly asked.
- If a CLI command fails, report the exact error rather than guessing.
