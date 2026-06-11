# Welcome to Databricks

Databricks is one place for all your data and AI work: store and govern data, run SQL and notebooks, build pipelines and dashboards, train and serve models, and ship apps. No stitching together separate tools.

This guide takes you from zero to your first command.

## What you need

A **workspace** — your Databricks home, at a URL like `https://your-company.cloud.databricks.com`. No access yet? Sign up for a free trial at https://www.databricks.com (Databricks Free Edition is great for learning). You've already got the CLI — it's how you're reading this.

## A few concepts (the 30-second version)

- **Workspace** — the environment you log into. Notebooks, jobs, dashboards, and data all live here.
- **Unity Catalog** — how Databricks governs your data. Tables are named in three parts: `catalog.schema.table` (think folder → subfolder → table).
- **Compute** — where your code runs. *SQL warehouses* run SQL; *clusters* and *serverless* run notebooks and jobs.
- **Profile** — a saved connection to a workspace on your machine, so you don't re-enter credentials each time.

## Step 1 — Connect to your workspace

```bash
databricks auth login
```
This opens **login.databricks.com** in your browser. Sign in, pick the workspace you want, and you're connected — no URL to look up. The CLI saves the connection as a *profile*; if it's your only workspace, press Enter to accept the suggested name and every command will use it automatically. Run `databricks auth login` again any time to add another workspace.

Already know your workspace URL? Go straight to it: `databricks auth login --host https://your-company.cloud.databricks.com --profile my-workspace`.

## Step 2 — Try it out

```bash
databricks current-user me     # who am I?
databricks catalogs list       # what data can I see?
databricks jobs list           # my jobs
```
To see everything a command can do, add `--help` — for example, `databricks jobs --help`.

Got more than one workspace? Name each at login (`databricks auth login --profile prod`), then target it with `--profile`, e.g. `databricks jobs list --profile prod`.

💡 **Tip:** turn on tab-completion so you can `Tab` through commands and flags: `databricks completion install`.

## Step 3 — Build something

The standard way to build on Databricks is **Databricks Asset Bundles (DABs)** — your jobs, pipelines, dashboards, and apps defined as code in one project, shipped with one command:

```bash
databricks bundle init                 # start from a template
databricks bundle validate             # check it
databricks bundle deploy -t dev        # ship it to your workspace
databricks bundle run <name> -t dev    # run a job or pipeline
```

Want a **full-stack app**? AppKit scaffolds one in TypeScript + React:

```bash
databricks apps init          # scaffold the app
databricks apps deploy        # deploy it to your workspace
databricks apps run-local     # develop locally (or dev-remote)
```

## Where to go next

- **Explore your data** — browse catalogs, schemas, and tables, or query them from the workspace UI.
- **More to build** — Lakeflow Jobs (orchestration), Lakeflow Pipelines (ETL), AI/BI Dashboards, and Model Serving.
- **Using an AI coding assistant?** Install the Databricks skills so it knows these patterns: `databricks aitools install`.

📚 Full docs: https://docs.databricks.com  •  CLI reference: https://docs.databricks.com/dev-tools/cli

Welcome aboard.
