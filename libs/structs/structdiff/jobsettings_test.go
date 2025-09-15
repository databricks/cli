package structdiff

import (
	"encoding/json"
	"testing"

	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// adapted from https://docs.databricks.com/api/workspace/jobs/get
const jobExampleResponse = `
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
        "task_key": "Match",
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

// Same example but values are zeroed
const jobExampleResponseZeroes = `
{
    "budget_policy_id": "",
    "continuous": {
      "pause_status": ""
    },
    "deployment": {
      "kind": "",
      "metadata_file_path": ""
    },
    "description": "",
    "edit_mode": "",
    "email_notifications": {
      "no_alert_for_skipped_runs": false,
      "on_duration_warning_threshold_exceeded": [
      ],
      "on_failure": [
      ],
      "on_start": [
      ],
      "on_streaming_backlog_exceeded": [
      ],
      "on_success": [
      ]
    },
    "environments": [
      {
        "environment_key": "",
        "spec": {
          "client": "",
          "dependencies": [
          ]
        }
      }
    ],
    "format": "",
    "git_source": {
      "git_branch": "",
      "git_provider": "",
      "git_url": ""
    },
    "health": {
      "rules": [
        {
          "metric": "",
          "op": "",
          "value": 0
        }
      ]
    },
    "job_clusters": [
      {
        "job_cluster_key": "",
        "new_cluster": {
          "autoscale": {
            "max_workers": 0,
            "min_workers": 0
          },
          "node_type_id": null,
          "spark_conf": {
            "spark.speculation": ""
          },
          "spark_version": ""
        }
      }
    ],
    "max_concurrent_runs": 0,
    "name": "",
    "notification_settings": {
      "no_alert_for_canceled_runs": false,
      "no_alert_for_skipped_runs": false
    },
    "parameters": [
      {
        "default": "",
        "name": ""
      }
    ],
    "performance_target": "",
    "queue": {
      "enabled": false
    },
    "run_as": {
      "service_principal_name": "",
      "user_name": ""
    },
    "schedule": {
      "pause_status": "",
      "quartz_cron_expression": "",
      "timezone_id": ""
    },
    "tags": {
      "cost-center": "",
      "team": ""
    },
    "tasks": [
      {
        "depends_on": [],
        "description": "",
        "existing_cluster_id": "",
        "libraries": [
        ],
        "max_retries": 0,
        "min_retry_interval_millis": 0,
        "retry_on_timeout": false,
        "spark_jar_task": {
          "main_class_name": "",
          "parameters": [
          ]
        },
        "task_key": "",
        "timeout_seconds": 0
      },
      {
        "depends_on": [],
        "description": "",
        "job_cluster_key": "",
        "libraries": [
          {
            "jar": ""
          }
        ],
        "max_retries": 0,
        "min_retry_interval_millis": 0,
        "retry_on_timeout": false,
        "spark_jar_task": {
          "main_class_name": "",
          "parameters": [
          ]
        },
        "task_key": "",
        "timeout_seconds": 0
      },
      {
        "depends_on": [
        ],
        "description": "",
        "max_retries": 0,
        "min_retry_interval_millis": 0,
        "new_cluster": {
          "autoscale": {
            "max_workers": 0,
            "min_workers": 0
          },
          "node_type_id": null,
          "spark_conf": {
            "spark.speculation": ""
          },
          "spark_version": ""
        },
        "notebook_task": {
          "base_parameters": {
          },
          "notebook_path": ""
        },
        "retry_on_timeout": false,
        "run_if": "",
        "task_key": "",
        "timeout_seconds": 0
      }
    ],
    "trigger": {
      "file_arrival": {
        "min_time_between_triggers_seconds": 0,
        "url": "",
        "wait_after_last_change_seconds": 0
      },
      "pause_status": "",
      "periodic": {
        "interval": 0,
        "unit": ""
      }
    }
}`

// Same example but every value is nil
const jobExampleResponseNils = `
{
    "deployment": {
    },
    "email_notifications": {
    },
    "environments": [
      {
      }
    ],
    "git_source": {

    },
    "health": {
      "rules": [
        {
        }
      ]
    },
    "job_clusters": [
      {
        "new_cluster": {
          "autoscale": {
          },
          "spark_conf": {
          }
        }
      }
    ],
    "notification_settings": {
    },
    "parameters": [
    ],
    "queue": {
    },
    "run_as": {
    },
    "schedule": {
    },
    "tags": {
    },
    "tasks": [
      {
      },
      {
      },
      {
      }
    ],
    "trigger": {
      "file_arrival": {
      },
      "periodic": {
      }
    }
}`

func testEqual(t *testing.T, input string) {
	var x, y jobs.JobSettings
	require.NoError(t, json.Unmarshal([]byte(input), &x))
	changes, err := GetStructDiff(x, x)
	require.NoError(t, err)
	require.Nil(t, changes)

	require.NoError(t, json.Unmarshal([]byte(input), &y))
	changes, err = GetStructDiff(x, y)
	require.NoError(t, err)
	require.Nil(t, changes)

	changes, err = GetStructDiff(&x, &y)
	require.NoError(t, err)
	require.Nil(t, changes)
}

func TestJobEqual(t *testing.T) {
	testEqual(t, jobExampleResponse)
	testEqual(t, jobExampleResponseZeroes)
	testEqual(t, jobExampleResponseNils)
}

func TestJobDiff(t *testing.T) {
	var job, zero, nils jobs.JobSettings

	require.NoError(t, json.Unmarshal([]byte(jobExampleResponse), &job))
	require.NoError(t, json.Unmarshal([]byte(jobExampleResponseZeroes), &zero))
	require.NoError(t, json.Unmarshal([]byte(jobExampleResponseNils), &nils))

	changes, err := GetStructDiff(job, zero)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(changes), 75)
	assert.Equal(t, ".budget_policy_id", changes[0].Path.String())
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", changes[0].Old)
	assert.Equal(t, "", changes[0].New)
	// Note: pause_status shows up as nil here because Continous does not have ForceSendFields field
	assert.Equal(t, ".continuous.pause_status", changes[1].Path.String())
	assert.Equal(t, jobs.PauseStatus("UNPAUSED"), changes[1].Old)
	assert.Nil(t, changes[1].New)
	assert.Equal(t, ".deployment.kind", changes[2].Path.String())
	assert.Equal(t, jobs.JobDeploymentKind("BUNDLE"), changes[2].Old)
	assert.Equal(t, jobs.JobDeploymentKind(""), changes[2].New)
	assert.Equal(t, ".deployment.metadata_file_path", changes[3].Path.String())
	assert.Equal(t, "string", changes[3].Old)
	assert.Equal(t, "", changes[3].New)

	changes, err = GetStructDiff(job, nils)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(changes), 77)
	assert.Equal(t, ".budget_policy_id", changes[0].Path.String())
	assert.Equal(t, "550e8400-e29b-41d4-a716-446655440000", changes[0].Old)
	assert.Nil(t, changes[0].New)

	// continous is completely deleted from jobExampleResponseNils
	assert.Equal(t, ".continuous", changes[1].Path.String())
	assert.Equal(t, &jobs.Continuous{PauseStatus: "UNPAUSED"}, changes[1].Old)
	assert.Nil(t, changes[1].New)

	// deployment.kind is not omitempty field, so it does not show up as nil here
	assert.Equal(t, ".deployment.kind", changes[2].Path.String())
	assert.Equal(t, jobs.JobDeploymentKind("BUNDLE"), changes[2].Old)
	assert.Equal(t, jobs.JobDeploymentKind(""), changes[2].New)

	assert.Equal(t, ".deployment.metadata_file_path", changes[3].Path.String())
	assert.Equal(t, "string", changes[3].Old)
	assert.Nil(t, changes[3].New)

	changes, err = GetStructDiff(zero, nils)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(changes), 58)
	assert.Equal(t, ".budget_policy_id", changes[0].Path.String())
	assert.Equal(t, "", changes[0].Old)
	assert.Nil(t, changes[0].New)
	assert.Equal(t, ".continuous", changes[1].Path.String())
	assert.Equal(t, &jobs.Continuous{}, changes[1].Old)
	assert.Nil(t, changes[1].New)

	// deployment.kind is "" in both

	assert.Equal(t, ".deployment.metadata_file_path", changes[2].Path.String())
	assert.Equal(t, "", changes[2].Old)
	assert.Nil(t, changes[2].New)
}
