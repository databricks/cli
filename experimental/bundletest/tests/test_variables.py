"""Variable resolution in the local backend.

Offline references (${var.name} from BUNDLE_VAR_* or a declared default) are resolved so
local config matches what deploy would render. References only the workspace can resolve
(${workspace.*}, a lookup variable, an unset variable) are left alone and rejected loudly
via LocalUnsupported at the use site — never silently passed through as the literal string.
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


def test_online_variable_in_sql_path_is_loud(tmp_path):
    b = _bundle(
        tmp_path,
        "variables:\n  q:\n    lookup:\n      warehouse: w\n"
        "resources:\n  jobs:\n    j:\n      tasks:\n        - task_key: t\n"
        "          sql_task:\n            file:\n              path: ${var.q}.sql\n",
    )
    with bundle_env(b) as env, pytest.raises(LocalUnsupported):
        env.run_job("j")
