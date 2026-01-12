#!/usr/bin/env python3
import argparse
import json
import copy

JOB_TEMPLATE_BASE = {
    "description": "This job contain multiple tasks that are required to produce the weekly shark sightings report.",
    "email_notifications": {
        "no_alert_for_skipped_runs": False,
        "on_failure": ["user.name@databricks.com"],
        "on_success": ["user.name@databricks.com"],
    },
    "job_clusters": [
        {
            "job_cluster_key": "auto_scaling_cluster",
            "new_cluster": {
                "autoscale": {"max_workers": 16, "min_workers": 2},
                "node_type_id": "i3.xlarge",
                "spark_conf": {"spark.speculation": "true"},
                "spark_version": "13.3.x-scala2.12",
            },
        }
    ],
    "max_concurrent_runs": 10,
    "name": "A multitask job",
    "notification_settings": {"no_alert_for_canceled_runs": False, "no_alert_for_skipped_runs": False},
    "parameters": [{"default": "users", "name": "table"}],
    "tags": {"cost-center": "engineering", "team": "jobs"},
    "tasks": [
        {
            "depends_on": [],
            "description": "Extracts session data from events",
            "job_cluster_key": "auto_scaling_cluster",
            "libraries": [{"jar": "dbfs:/mnt/databricks/Sessionize.jar"}],
            "max_retries": 3,
            "min_retry_interval_millis": 2000,
            "retry_on_timeout": False,
            "spark_jar_task": {
                "main_class_name": "com.databricks.Sessionize",
                "parameters": ["--data", "dbfs:/path/to/data.json"],
            },
            "task_key": "Sessionize",
            "timeout_seconds": 86400,
        },
        {
            "depends_on": [],
            "description": "Ingests order data",
            "job_cluster_key": "auto_scaling_cluster",
            "libraries": [{"jar": "dbfs:/mnt/databricks/OrderIngest.jar"}],
            "max_retries": 3,
            "min_retry_interval_millis": 2000,
            "retry_on_timeout": False,
            "spark_jar_task": {
                "main_class_name": "com.databricks.OrdersIngest",
                "parameters": ["--data", "dbfs:/path/to/order-data.json"],
            },
            "task_key": "Orders_Ingest",
            "timeout_seconds": 86400,
        },
        {
            "depends_on": [{"task_key": "Orders_Ingest"}, {"task_key": "Sessionize"}],
            "description": "Matches orders with user sessions",
            "max_retries": 3,
            "min_retry_interval_millis": 2000,
            "new_cluster": {
                "autoscale": {"max_workers": 16, "min_workers": 2},
                "node_type_id": "i3.xlarge",
                "spark_conf": {"spark.speculation": "true"},
                "spark_version": "13.3.x-scala2.12",
            },
            "notebook_task": {
                "base_parameters": {"age": "35", "name": "John Doe"},
                "notebook_path": "/Users/user.name@databricks.com/Match",
            },
            "retry_on_timeout": False,
            "run_if": "ALL_SUCCESS",
            "task_key": "Match",
            "timeout_seconds": 86400,
        },
    ],
    "timeout_seconds": 86400,
}


def gen_config(n):
    jobs = {}
    for i in range(n):
        job = copy.deepcopy(JOB_TEMPLATE_BASE)
        job["name"] = f"job_{i}"

        # Odd jobs use continuous, even jobs use schedule
        if i % 2 == 1:
            job["continuous"] = {"pause_status": "UNPAUSED"}
        else:
            job["schedule"] = {
                "pause_status": "UNPAUSED",
                "quartz_cron_expression": "20 30 * * * ?",
                "timezone_id": "Europe/London",
            }

        jobs[f"job_{i}"] = job

    config = {"bundle": {"name": "test-bundle"}, "resources": {"jobs": jobs}}

    return config


def print_yaml(obj, indent=0, list_item=False):
    indent_str = "  " * indent

    if isinstance(obj, dict):
        first = True
        for key, value in obj.items():
            if list_item and first:
                prefix = indent_str + "- "
                first = False
            elif list_item:
                prefix = indent_str + "  "
            else:
                prefix = indent_str
            nested_indent = indent + 2 if list_item else indent + 1
            if isinstance(value, (dict, list)) and value:
                print(f"{prefix}{key}:")
                print_yaml(value, nested_indent)
            else:
                print(f"{prefix}{key}: {json.dumps(value)}")
    elif isinstance(obj, list):
        for item in obj:
            if isinstance(item, (dict, list)):
                print_yaml(item, indent, list_item=True)
            else:
                print(f"{indent_str}- {json.dumps(item)}")
    else:
        prefix = f"{indent_str}- " if list_item else indent_str
        print(f"{prefix}{json.dumps(obj)}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--jobs", type=int, default=10, help="Number of jobs to generate")
    args = parser.parse_args()

    print_yaml(gen_config(args.jobs))


if __name__ == "__main__":
    main()
