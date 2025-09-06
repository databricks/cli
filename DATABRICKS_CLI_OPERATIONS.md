# Databricks CLI Operations Guide

This guide covers essential operations for managing Databricks Jobs and executing SQL statements using the Databricks CLI.


### List Jobs

**Tool:** `databricks jobs list`
**Description:** List all jobs in the workspace with their basic information including job ID, name, and settings.

**Example CLI Usage:**
```bash
# List all jobs
databricks jobs list

# List jobs with detailed output
databricks jobs list --output json

# List jobs with custom output format
databricks jobs list --output json | jq '.[] | {id: .job_id, name: .settings.name}'
```

### List Jobs by Username

**Tool:** `databricks jobs list` with filtering
**Description:** Filter jobs by creator username to find jobs created by specific users.

**Example CLI Usage:**
```bash
# List jobs created by a specific user (requires API filtering)
databricks jobs list --output json | jq '.[] | select(.creator_user_name == "john.doe@databricks.com")'

# Alternative: Use workspace search for user-specific jobs
databricks workspace list /Users/john.doe@databricks.com/ | grep -i job

# Get detailed job information for a specific user's jobs
databricks jobs list --output json | jq '.[] | select(.creator_user_name | contains("john.doe")) | {job_id, name: .settings.name, creator: .creator_user_name}'
```

### Create Job

**Tool:** `databricks jobs create`
**Description:** Create a new Databricks job with specified configuration including tasks, clusters, schedules, and parameters.

**Example CLI Usage:**
```bash
# Create a simple notebook job
databricks jobs create --json '{
  "name": "Daily ETL Pipeline",
  "tasks": [
    {
      "task_key": "extract",
      "notebook_task": {
        "notebook_path": "/Workspace/Shared/ETL/extract",
        "source": "WORKSPACE"
      },
      "job_cluster_key": "extract_cluster"
    }
  ],
  "job_clusters": [
    {
      "job_cluster_key": "extract_cluster",
      "new_cluster": {
        "spark_version": "13.3.x-scala2.12",
        "node_type_id": "Standard_DS3_v2",
        "num_workers": 2
      }
    }
  ],
  "schedule": {
    "quartz_cron_expression": "0 0 6 * * ?",
    "timezone_id": "America/New_York"
  }
}'

# Create a job with multiple tasks
databricks jobs create --json '{
  "name": "Multi-Task Job",
  "tasks": [
    {
      "task_key": "task1",
      "notebook_task": {"notebook_path": "/Workspace/notebooks/task1"},
      "depends_on": []
    },
    {
      "task_key": "task2",
      "notebook_task": {"notebook_path": "/Workspace/notebooks/task2"},
      "depends_on": [{"task_key": "task1"}]
    }
  ],
  "job_clusters": [
    {
      "job_cluster_key": "shared_cluster",
      "new_cluster": {
        "spark_version": "13.3.x-scala2.12",
        "node_type_id": "Standard_DS3_v2",
        "autoscale": {"min_workers": 1, "max_workers": 4}
      }
    }
  ]
}'
```

### Update Job

**Tool:** `databricks jobs update`
**Description:** Update an existing job's configuration, including tasks, clusters, schedules, and parameters.

**Example CLI Usage:**
```bash
# Update job schedule
databricks jobs update 1234567890 --json '{
  "fields_to_remove": ["schedule"],
  "schedule": {
    "quartz_cron_expression": "0 30 7 * * ?",
    "timezone_id": "America/New_York"
  }
}'

# Update job cluster configuration
databricks jobs update 1234567890 --json '{
  "job_clusters": [
    {
      "job_cluster_key": "extract_cluster",
      "new_cluster": {
        "spark_version": "14.3.x-scala2.12",
        "node_type_id": "Standard_DS4_v2",
        "num_workers": 4
      }
    }
  ]
}'

# Add parameters to a job
databricks jobs update 1234567890 --json '{
  "parameters": [
    {
      "name": "environment",
      "default": "production"
    },
    {
      "name": "input_date",
      "default": "2024-01-01"
    }
  ]
}'

# Update job name and description
databricks jobs update 1234567890 --json '{
  "name": "Updated ETL Pipeline",
  "description": "Daily ETL pipeline with improved performance"
}'
```

### Run Job

**Tool:** `databricks jobs run-now`
**Description:** Trigger an immediate execution of a job with optional parameters.

**Example CLI Usage:**
```bash
# Run job immediately
databricks jobs run-now 1234567890

# Run job with parameters
databricks jobs run-now 1234567890 --json '{
  "job_parameters": {
    "environment": "staging",
    "input_date": "2024-01-15"
  }
}'

# Run job with specific cluster
databricks jobs run-now 1234567890 --json '{
  "job_parameters": {"env": "prod"},
  "job_cluster_key": "production_cluster"
}'

# Run job and capture run ID for monitoring
RUN_ID=$(databricks jobs run-now 1234567890 --output json | jq -r '.run_id')
echo "Job run started with ID: $RUN_ID"
```

### Get Job Status

**Tool:** `databricks jobs runs get`
**Description:** Get the status and details of a specific job run.

**Example CLI Usage:**
```bash
# Get job run status
databricks jobs runs get 1234567890

# Get detailed run information
databricks jobs runs get 1234567890 --output json

# Monitor job run in a loop
RUN_ID=1234567890
while true; do
  STATUS=$(databricks jobs runs get $RUN_ID --output json | jq -r '.state.life_cycle_state')
  echo "Job status: $STATUS"
  if [[ "$STATUS" == "TERMINATED" ]] || [[ "$STATUS" == "SKIPPED" ]] || [[ "$STATUS" == "INTERNAL_ERROR" ]]; then
    break
  fi
  sleep 30
done
```

### Cancel Job Run

**Tool:** `databricks jobs runs cancel`
**Description:** Cancel a running job execution.

**Example CLI Usage:**
```bash
# Cancel a running job
databricks jobs runs cancel 1234567890

# Cancel with confirmation
databricks jobs runs cancel 1234567890 --yes

# Cancel multiple runs
for RUN_ID in 1234567890 1234567891; do
  databricks jobs runs cancel $RUN_ID
done
```

## SQL Statement Execution

### Execute SQL Statement

**Tool:** `databricks statement-execution execute-statement`
**Description:** Execute SQL statements against Databricks SQL warehouses with support for various output formats and execution modes.

**Example CLI Usage:**
```bash
# Execute simple query (uses default warehouse a5e694153a0d5e8c)
databricks statement-execution execute-statement "SELECT * FROM main.gold_mls.search_listings LIMIT 5"

```
