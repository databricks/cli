# Apps-MCP Evals

Databricks Asset Bundle for generating and evaluating apps using the Apps-MCP system with klaudbiusz framework.

## Overview

This bundle provides two jobs:
1. **Generation Job** - Generates apps using klaudbiusz with the Databricks CLI as MCP server
2. **Evaluation Job** - Evaluates generated apps and logs results to MLflow

## Prerequisites

1. **Databricks Secrets** - Create secret scope and add tokens:
   ```bash
   databricks secrets create-scope apps-mcp-evals
   databricks secrets put-secret apps-mcp-evals anthropic-api-key
   databricks secrets put-secret apps-mcp-evals databricks-token
   ```

2. **UC Volumes** - Create volumes for artifacts:
   ```bash
   databricks volumes create main.default.apps_mcp_artifacts
   databricks volumes create main.default.apps_mcp_generated
   ```

3. **CLI Binary** - Build and upload Linux CLI binary:
   ```bash
   GOOS=linux GOARCH=amd64 go build -o databricks-linux
   databricks fs cp databricks-linux /Volumes/main/default/apps_mcp_artifacts/
   ```

## Quick Start

```bash
cd experimental/apps-mcp/evals

# Validate bundle
databricks bundle validate -t dev

# Deploy
databricks bundle deploy -t dev

# Run generation (creates apps in UC Volume)
databricks bundle run -t dev apps_generation_job

# Run evaluation (evaluates apps, logs to MLflow)
databricks bundle run -t dev apps_eval_job
```

## Jobs

### Generation Job (`apps_generation_job`)

Generates apps using klaudbiusz's local_run with LiteLLM backend.

**Parameters:**
- `prompts` - Prompt set: `databricks`, `databricks_v2`, or `test` (default: `test`)
- `cli_binary_volume` - Path to CLI binary volume
- `apps_volume` - Output volume for generated apps

**Cluster:** Jobs cluster with Spark 16.2.x (Python 3.12)

### Evaluation Job (`apps_eval_job`)

Evaluates generated apps using klaudbiusz's Docker-based evaluation.

**Parameters:**
- `apps_volume` - Volume containing apps to evaluate
- `mlflow_experiment` - MLflow experiment for logging results
- `parallelism` - Number of parallel evaluations

**Cluster:** Jobs cluster with Spark 16.2.x, Docker installed via init script

**Schedule:** Nightly at 2am UTC

## Configuration

### Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `prompts` | Prompt set for generation | `test` |
| `cli_binary_volume` | UC Volume for CLI binary | `/Volumes/main/default/apps_mcp_artifacts` |
| `apps_volume` | UC Volume for generated apps | `/Volumes/main/default/apps_mcp_generated` |
| `mlflow_experiment` | MLflow experiment path | `/Shared/apps-mcp-evaluations` |
| `eval_parallelism` | Parallel eval workers | `4` |
| `evals_git_url` | klaudbiusz repo URL | `https://github.com/neondatabase/appdotbuild-agent.git` |

### Targets

- **dev** - Development mode, staging MLflow experiment
- **prod** - Production mode, service principal identity

## Monitoring

- **MLflow** - View metrics at the configured experiment path
- **Health Alerts** - Eval job alerts if runtime exceeds 2 hours
- **Logs** - Check job run output for detailed evaluation results

## Architecture

```
evals/
├── databricks.yml              # Bundle configuration
├── resources/
│   ├── apps_generation_job.job.yml  # Generation job
│   └── apps_eval_job.job.yml        # Evaluation job
├── init/
│   ├── setup_generation.sh     # Generation cluster init
│   └── setup_eval.sh           # Eval cluster init (Docker)
├── src/
│   ├── generate_apps.py        # App generation orchestrator
│   └── run_evals.py            # Evaluation orchestrator
└── pyproject.toml              # Python package config
```

## Prompt Sets

Available prompt sets (configured via `prompts` variable):

- `test` - Simple test prompts (1 app) for quick validation
- `databricks` - 5 Databricks-focused dashboard prompts
- `databricks_v2` - 20 realistic human-style prompts

## Known Limitations

- Docker containers require `--privileged` flag on Databricks clusters
- Generation uses LiteLLM backend (Claude Agent SDK has root user restriction)
- UC Volumes don't support symlinks, uses `latest.txt` file instead
