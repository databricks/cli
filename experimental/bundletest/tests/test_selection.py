from pathlib import Path

from bundletest.selection import format_change_selection, select_changes


def _write_bundle(root: Path) -> None:
    (root / "databricks.yml").write_text(
        """resources:
  jobs:
    transform:
      tasks:
        - task_key: transform
          sql_task:
            file:
              path: src/transform.sql
    aggregate:
      tasks:
        - task_key: aggregate
          sql_task:
            file:
              path: src/aggregate.sql
  dashboards:
    overview:
      file_path: dashboards/overview.json
"""
    )


def test_changed_source_selects_referencing_resource(tmp_path):
    _write_bundle(tmp_path)

    selection = select_changes(tmp_path, ["src/transform.sql"])

    assert selection.resources == ("jobs.transform",)
    assert not selection.run_all
    assert "[RESOURCE] jobs.transform" in format_change_selection(selection, "main")


def test_changed_directory_selects_resource(tmp_path):
    _write_bundle(tmp_path)
    (tmp_path / "app").mkdir()
    config = (tmp_path / "databricks.yml").read_text()
    (tmp_path / "databricks.yml").write_text(config + "  apps:\n    portal:\n      source_code_path: app\n")

    selection = select_changes(tmp_path, ["app/server.py"])

    assert selection.resources == ("apps.portal",)


def test_changed_test_is_selected_directly(tmp_path):
    _write_bundle(tmp_path)
    test = tmp_path / "tests" / "test_transform.py"

    selection = select_changes(tmp_path, ["tests/test_transform.py"])

    assert selection.test_files == (test.resolve().as_posix(),)


def test_changed_yaml_runs_complete_suite(tmp_path):
    _write_bundle(tmp_path)

    selection = select_changes(tmp_path, ["resources/jobs.yml"])

    assert selection.run_all
    assert set(selection.resources) == {"jobs.transform", "jobs.aggregate", "dashboards.overview"}


def test_changed_source_resolves_resources_from_included_files(tmp_path):
    (tmp_path / "databricks.yml").write_text("bundle:\n  name: demo\ninclude:\n  - resources/*.yml\n")
    resources = tmp_path / "resources"
    resources.mkdir()
    (resources / "jobs.yml").write_text(
        "resources:\n  jobs:\n    transform:\n      tasks:\n"
        "        - task_key: transform\n          sql_task:\n"
        "            file:\n              path: src/transform.sql\n"
    )

    selection = select_changes(tmp_path, ["src/transform.sql"])

    assert selection.resources == ("jobs.transform",)
