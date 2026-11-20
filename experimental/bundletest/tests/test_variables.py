"""Config resolution in the local backend.

Resolution reuses the CLI's own offline engine, so includes, target overrides, and offline
references (${var.name} from BUNDLE_VAR_* or a declared default) resolve exactly as deploy
would render them. References only the workspace can resolve (${workspace.*}, a lookup
variable, an unset variable) are rejected loudly via LocalUnsupported at the use site —
never silently passed through.
"""

import pytest
from bundletest import LocalUnsupported, bundle_env


def _bundle(tmp_path, yml: str) -> str:
    (tmp_path / "databricks.yml").write_text(yml)
    return str(tmp_path)


def test_default_is_resolved(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  catalog:\n    default: shop\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.catalog}_job\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env:
        assert env.backend.get_resource("jobs", "j")["name"] == "shop_job"


def test_env_override_wins(tmp_path, monkeypatch):
    monkeypatch.setenv("BUNDLE_VAR_catalog", "prod")
    b = _bundle(
        tmp_path,
        "variables:\n  catalog:\n    default: shop\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.catalog}_job\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env:
        assert env.backend.get_resource("jobs", "j")["name"] == "prod_job"


def test_nested_variable_is_resolved(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  env:\n    default: dev\n  catalog:\n    default: shop_${var.env}\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.catalog}\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env:
        assert env.backend.get_resource("jobs", "j")["name"] == "shop_dev"


def test_workspace_reference_is_loud(tmp_path):
    b = _bundle(
        tmp_path,
        "resources:\n  jobs:\n    j:\n      name: job\n"
        "      run_as:\n        user_name: ${workspace.current_user.userName}\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env, pytest.raises(LocalUnsupported):
        env.backend.get_resource("jobs", "j")


def test_lookup_variable_is_loud(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  wh:\n    lookup:\n      warehouse: my-warehouse\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.wh}\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env, pytest.raises(LocalUnsupported):
        env.backend.get_resource("jobs", "j")


def test_unset_variable_is_loud(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  warehouse_id:\n    description: set at deploy time\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.warehouse_id}\n"
        "      tasks:\n        - task_key: t\n",
    )
    with bundle_env(b) as env, pytest.raises(LocalUnsupported):
        env.backend.get_resource("jobs", "j")


def test_included_file_is_resolved(tmp_path):
    # A resource defined in an included file, referencing a variable. The old single-file
    # resolver never read includes, so it couldn't see this job at all.
    (tmp_path / "databricks.yml").write_text(
        "bundle:\n  name: b\ninclude:\n  - resources/*.yml\n"
        "variables:\n  catalog:\n    default: shop\n"
    )
    (tmp_path / "resources").mkdir()
    (tmp_path / "resources" / "jobs.yml").write_text(
        "resources:\n  jobs:\n    j:\n      name: ${var.catalog}_job\n"
        "      tasks:\n        - task_key: t\n"
    )
    with bundle_env(str(tmp_path)) as env:
        assert env.backend.get_resource("jobs", "j")["name"] == "shop_job"


def test_target_override_is_resolved(tmp_path):
    # The default target overrides a variable. The old resolver ignored targets and would
    # have rendered the base default ("base_job").
    b = _bundle(
        tmp_path,
        "bundle:\n  name: b\n"
        "variables:\n  catalog:\n    default: base\n"
        "resources:\n  jobs:\n    j:\n      name: ${var.catalog}_job\n"
        "      tasks:\n        - task_key: t\n"
        "targets:\n  dev:\n    default: true\n    variables:\n      catalog: devcat\n",
    )
    with bundle_env(b) as env:
        assert env.backend.get_resource("jobs", "j")["name"] == "devcat_job"


def test_online_variable_in_sql_path_is_loud(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  q:\n    lookup:\n      warehouse: w\n"
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          sql_task:\n            file:\n              path: ${var.q}.sql\n",
    )
    with bundle_env(b) as env, pytest.raises(LocalUnsupported):
        env.run_job("j")
