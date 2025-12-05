# Apps-MCP Continuous Evals

Databricks Asset Bundle for running continuous evaluations of the Apps-MCP code generation system.

## Overview

This bundle deploys a scheduled Databricks job that:
1. Runs the klaudbiusz evaluation framework
2. Logs results to MLflow for tracking
3. Alerts on failures or long-running evaluations

## Quick Start

```bash
# Validate the bundle
databricks bundle validate -t dev

# Deploy to dev workspace
databricks bundle deploy -t dev

# Run manually
databricks bundle run -t dev apps_eval_job

# View results in MLflow
# Navigate to: ML → Experiments → /Shared/apps-mcp-evaluations-staging
```

## Configuration

### Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `catalog` | Unity Catalog for results | `main` |
| `schema` | Schema for eval tables | `${workspace.current_user.short_name}` (dev) |
| `mlflow_experiment` | MLflow experiment path | `/Shared/apps-mcp-evaluations` |
| `eval_parallelism` | Parallel eval workers | `4` |

### Targets

- **dev**: Development mode with personal schema, staging MLflow experiment
- **prod**: Production mode with shared schema, service principal identity

## Schedule

The job runs nightly at 2am UTC. Manual runs can be triggered via:

```bash
databricks bundle run -t dev apps_eval_job
```

## Monitoring

- **MLflow**: View metrics trends at `/Shared/apps-mcp-evaluations`
- **Health Alerts**: Job alerts if runtime exceeds 2 hours
- **Email**: Failures notify apps-mcp-team@databricks.com

## Development

```bash
# Build wheel locally
uv build --wheel

# Run evals locally (outside Databricks)
uv run python -m src.run_evals --mode=eval_only --parallelism=4
```

## Architecture

```
evals/
├── databricks.yml           # Bundle configuration
├── resources/
│   └── apps_eval_job.job.yml  # Job definition
├── src/
│   ├── __init__.py
│   └── run_evals.py         # Main orchestrator
├── pyproject.toml           # Python package config
└── README.md
```
