import pytest

from databricks.bundles.jobs import JobPermission
from databricks.bundles.jobs._models.job_permission_level import JobPermissionLevel


def test_oneof_one():
    permission = JobPermission(
        level=JobPermissionLevel.CAN_VIEW,
        user_name="test@example.com",
    )

    assert permission


def test_oneof_none():
    with pytest.raises(ValueError) as exc_info:
        JobPermission(level=JobPermissionLevel.CAN_VIEW)

    assert exc_info.exconly() == (
        "ValueError: JobPermission must specify exactly one of 'user_name', "
        "'service_principal_name', 'group_name'"
    )


def test_oneof_both():
    with pytest.raises(ValueError) as exc_info:
        JobPermission(
            level=JobPermissionLevel.CAN_VIEW,
            user_name="test@example.com",
            service_principal_name="secret",
        )

    assert exc_info.exconly() == (
        "ValueError: JobPermission must specify exactly one of 'user_name', "
        "'service_principal_name', 'group_name'"
    )
