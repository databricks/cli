from pathlib import Path

import pytest
from bundletest.cli import main


def _write_bundle(root: Path) -> None:
    (root / "tests").mkdir()
    (root / "databricks.yml").write_text("bundle:\n  name: demo\nresources: {}\n")


def test_run_uses_local_backend_and_default_test_directory(tmp_path, monkeypatch):
    _write_bundle(tmp_path)
    captured = []
    original = Path.cwd()

    def run_pytest(args):
        assert Path.cwd() == tmp_path
        captured.extend(args)
        return 0

    monkeypatch.setattr(pytest, "main", run_pytest)

    assert main(["--bundle", str(tmp_path), "--local", "-q"]) == 0

    assert captured == ["-q", str(tmp_path / "tests")]
    assert Path.cwd() == original


def test_pytest_option_value_is_not_mistaken_for_test_path(tmp_path, monkeypatch):
    _write_bundle(tmp_path)
    captured = []
    monkeypatch.setattr(pytest, "main", lambda args: captured.extend(args) or 0)

    assert main(["--bundle", str(tmp_path), "-p", "no:cacheprovider"]) == 0

    assert captured == ["-p", "no:cacheprovider", str(tmp_path / "tests")]


def test_cloud_requires_explicit_profile(tmp_path, capsys):
    _write_bundle(tmp_path)

    assert main(["--bundle", str(tmp_path), "--cloud"]) == 2

    assert "--cloud requires an explicit --profile" in capsys.readouterr().err


def test_invalid_bundle_yaml_has_concise_error(tmp_path, capsys):
    (tmp_path / "databricks.yml").write_text("resources: [")

    assert main(["--bundle", str(tmp_path)]) == 2

    error = capsys.readouterr().err
    assert "bundletest: offline bundle resolution failed" in error
    assert "Traceback" not in error


def test_local_rejects_cloud_options(tmp_path, capsys):
    _write_bundle(tmp_path)

    assert main(["--bundle", str(tmp_path), "--local", "--profile", "dev"]) == 2

    assert "--profile can only be used with --cloud" in capsys.readouterr().err


def test_support_only_does_not_invoke_pytest(tmp_path, monkeypatch):
    _write_bundle(tmp_path)
    monkeypatch.setattr(pytest, "main", lambda args: pytest.fail("pytest should not run"))

    assert main(["--bundle", str(tmp_path), "--support-only"]) == 0


@pytest.mark.parametrize(
    ("arguments", "message"),
    [
        (["--base", "origin/main"], "--base requires --changed"),
        (
            ["--support-only", "--no-support-report"],
            "--support-only cannot be used with --no-support-report",
        ),
    ],
)
def test_rejects_incompatible_options(tmp_path, capsys, arguments, message):
    _write_bundle(tmp_path)

    assert main(["--bundle", str(tmp_path), *arguments]) == 2

    assert message in capsys.readouterr().err


def test_init_generates_a_starter(tmp_path, capsys):
    _write_bundle(tmp_path)
    (tmp_path / "databricks.yml").write_text(
        """resources:
  jobs:
    transform:
      tasks:
        - task_key: transform
          sql_task:
            file:
              path: transform.sql
"""
    )

    assert main(["init", str(tmp_path)]) == 0

    assert (tmp_path / "tests" / "test_bundle.py").is_file()
    assert "next: bundletest --local" in capsys.readouterr().out
