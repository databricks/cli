from pathlib import Path

from bundletest.support import format_support_report, inspect_local_support


def _write_bundle(root: Path) -> None:
    (root / "src").mkdir()
    (root / "src" / "local.sql").write_text("SELECT * FROM shop.bronze.orders")
    (root / "src" / "cloud.sql").write_text("SELECT * FROM main.bronze.orders")
    (root / "databricks.yml").write_text(
        """bundle:
  name: demo
resources:
  jobs:
    local_job:
      tasks:
        - task_key: local
          sql_task:
            file:
              path: src/local.sql
    cloud_job:
      tasks:
        - task_key: cloud
          sql_task:
            file:
              path: src/cloud.sql
    notebook_job:
      tasks:
        - task_key: notebook
          notebook_task:
            notebook_path: src/notebook.py
  dashboards:
    overview:
      file_path: dashboard.json
"""
    )


def test_support_report_classifies_local_and_cloud_tasks(tmp_path):
    _write_bundle(tmp_path)

    report = inspect_local_support(tmp_path)

    assert report.bundle_name == "demo"
    assert [(entry.resource, entry.status) for entry in report.entries] == [
        ("jobs.local_job", "local"),
        ("jobs.cloud_job", "cloud"),
        ("jobs.notebook_job", "cloud"),
    ]
    assert report.config_only_resources == 1
    rendered = format_support_report(tmp_path, report)
    assert "[LOCAL ] jobs.local_job/local: runs src/local.sql" in rendered
    assert "catalog 'main' is reserved by DuckDB" in rendered
    assert "summary: 1 local, 2 cloud-only, 1 config-only" in rendered


def test_support_report_surfaces_missing_sql_file(tmp_path):
    (tmp_path / "databricks.yml").write_text(
        """resources:
  jobs:
    broken:
      tasks:
        - task_key: missing
          sql_task:
            file:
              path: src/missing.sql
"""
    )

    report = inspect_local_support(tmp_path)

    assert report.entries[0].status == "error"
    assert "does not exist" in report.entries[0].reason
    assert "summary: 0 local, 0 cloud-only, 0 config-only, 1 error" in format_support_report(tmp_path, report)


def test_support_report_counts_job_without_tasks_as_config_only(tmp_path):
    (tmp_path / "databricks.yml").write_text("resources:\n  jobs:\n    empty: {}\n")

    report = inspect_local_support(tmp_path)

    assert report.config_count == 1
    assert "summary: 0 local, 0 cloud-only, 1 config-only" in format_support_report(tmp_path, report)
