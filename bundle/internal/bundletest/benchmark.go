package bundletest

import (
	"context"
	"encoding/json"
	"strconv"
	"testing"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/require"
)

const jobExample = `
{
    "budget_policy_id": "550e8400-e29b-41d4-a716-446655440000",
    "continuous": {
      "pause_status": "UNPAUSED"
    },
    "deployment": {
      "kind": "BUNDLE",
      "metadata_file_path": "string"
    },
    "description": "This job contain multiple tasks that are required to produce the weekly shark sightings report.",
    "edit_mode": "UI_LOCKED",
    "email_notifications": {
      "no_alert_for_skipped_runs": false,
      "on_duration_warning_threshold_exceeded": [
        "user.name@databricks.com"
      ],
      "on_failure": [
        "user.name@databricks.com"
      ],
      "on_start": [
        "user.name@databricks.com"
      ],
      "on_streaming_backlog_exceeded": [
        "user.name@databricks.com"
      ],
      "on_success": [
        "user.name@databricks.com"
      ]
    },
    "environments": [
      {
        "environment_key": "string",
        "spec": {
          "client": "1",
          "dependencies": [
            "string"
          ]
        }
      }
    ],
    "format": "SINGLE_TASK",
    "git_source": {
      "git_branch": "main",
      "git_provider": "gitHub",
      "git_url": "https://github.com/databricks/databricks-cli"
    },
    "health": {
      "rules": [
        {
          "metric": "RUN_DURATION_SECONDS",
          "op": "GREATER_THAN",
          "value": 10
        }
      ]
    },
    "job_clusters": [
      {
        "job_cluster_key": "auto_scaling_cluster",
        "new_cluster": {
          "autoscale": {
            "max_workers": 16,
            "min_workers": 2
          },
          "node_type_id": null,
          "spark_conf": {
            "spark.speculation": "true"
          },
          "spark_version": "7.3.x-scala2.12"
        }
      }
    ],
    "max_concurrent_runs": 10,
    "name": "A multitask job",
    "notification_settings": {
      "no_alert_for_canceled_runs": false,
      "no_alert_for_skipped_runs": false
    },
    "parameters": [
      {
        "default": "users",
        "name": "table"
      }
    ],
    "performance_target": "PERFORMANCE_OPTIMIZED",
    "queue": {
      "enabled": true
    },
    "run_as": {
      "service_principal_name": "692bc6d0-ffa3-11ed-be56-0242ac120002",
      "user_name": "user@databricks.com"
    },
    "schedule": {
      "pause_status": "UNPAUSED",
      "quartz_cron_expression": "20 30 * * * ?",
      "timezone_id": "Europe/London"
    },
    "tags": {
      "cost-center": "engineering",
      "team": "jobs"
    },
    "tasks": [
      {
        "depends_on": [],
        "description": "Extracts session data from events",
        "existing_cluster_id": "0923-164208-meows279",
        "libraries": [
          {
            "jar": "dbfs:/mnt/databricks/Sessionize.jar"
          }
        ],
        "max_retries": 3,
        "min_retry_interval_millis": 2000,
        "retry_on_timeout": false,
        "spark_jar_task": {
          "main_class_name": "com.databricks.Sessionize",
          "parameters": [
            "--data",
            "dbfs:/path/to/data.json"
          ]
        },
        "task_key": "Sessionize",
        "timeout_seconds": 86400
      },
      {
        "depends_on": [],
        "description": "Ingests order data",
        "job_cluster_key": "auto_scaling_cluster",
        "libraries": [
          {
            "jar": "dbfs:/mnt/databricks/OrderIngest.jar"
          }
        ],
        "max_retries": 3,
        "min_retry_interval_millis": 2000,
        "retry_on_timeout": false,
        "spark_jar_task": {
          "main_class_name": "com.databricks.OrdersIngest",
          "parameters": [
            "--data",
            "dbfs:/path/to/order-data.json"
          ]
        },
        "task_key": "Orders_Ingest",
        "timeout_seconds": 86400
      },
      {
        "depends_on": [
          {
            "task_key": "Orders_Ingest"
          },
          {
            "task_key": "Sessionize"
          }
        ],
        "description": "Matches orders with user sessions",
        "max_retries": 3,
        "min_retry_interval_millis": 2000,
        "new_cluster": {
          "autoscale": {
            "max_workers": 16,
            "min_workers": 2
          },
          "node_type_id": null,
          "spark_conf": {
            "spark.speculation": "true"
          },
          "spark_version": "7.3.x-scala2.12"
        },
        "notebook_task": {
          "base_parameters": {
            "age": "35",
            "name": "John Doe"
          },
          "notebook_path": "/Users/user.name@databricks.com/Match"
        },
        "retry_on_timeout": false,
        "run_if": "ALL_SUCCESS",
        "timeout_seconds": 86400
      }
    ],
    "timeout_seconds": 86400,
    "trigger": {
      "file_arrival": {
        "min_time_between_triggers_seconds": 0,
        "url": "string",
        "wait_after_last_change_seconds": 0
      },
      "pause_status": "UNPAUSED",
      "periodic": {
        "interval": 0,
        "unit": "HOURS"
      }
    }
}`

func BundleV(b *testing.B, numJobs int) dyn.Value {
	allJobs := map[string]*resources.Job{}
	for i := range numJobs {
		job := jobs.JobSettings{}
		err := json.Unmarshal([]byte(jobExample), &job)
		require.NoError(b, err)

		allJobs[strconv.Itoa(i)] = &resources.Job{
			JobSettings: job,
		}
	}

	myBundle := bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: allJobs,
			},
		},
	}

	// Apply noop mutator to initialize the bundle value.
	bundle.ApplyFuncContext(context.Background(), &myBundle, func(ctx context.Context, b *bundle.Bundle) {})

	return myBundle.Config.Value()
}

func Bundle(b *testing.B, numJobs int) *bundle.Bundle {
	allJobs := map[string]*resources.Job{}
	for i := range numJobs {
		job := jobs.JobSettings{}
		err := json.Unmarshal([]byte(jobExample), &job)
		require.NoError(b, err)

		allJobs[strconv.Itoa(i)] = &resources.Job{
			JobSettings: job,
		}
	}

	myBundle := bundle.Bundle{
		Config: config.Root{
			Resources: config.Resources{
				Jobs: allJobs,
			},
		},
	}

	// Apply noop mutator to initialize the bundle value.
	bundle.ApplyFuncContext(context.Background(), &myBundle, func(ctx context.Context, b *bundle.Bundle) {})

	return &myBundle
}
