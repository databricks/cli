from pathlib import Path

import pytest
from bundletest.scaffold import create_starter


def _write_bundle(root: Path) -> None:
    (root / "databricks.yml").write_text(
        """resources:
  jobs:
    transform_orders:
      tasks:
        - task_key: transform
          sql_task:
            file:
              path: src/transform.sql
"""
    )


def test_create_starter_generates_fixture_and_marked_test(tmp_path):
    _write_bundle(tmp_path)

    generated = create_starter(tmp_path)

    assert generated.resource == "jobs.transform_orders"
    assert (tmp_path / "tests" / "conftest.py").is_file()
    test = (tmp_path / "tests" / "test_bundle.py").read_text()
    assert 'pytest.mark.bundle_resource("jobs.transform_orders")' in test
    assert 'env.jobs["transform_orders"]' in test


def test_create_starter_does_not_overwrite_tests_by_default(tmp_path):
    _write_bundle(tmp_path)
    create_starter(tmp_path)

    with pytest.raises(FileExistsError, match="--force"):
        create_starter(tmp_path)
