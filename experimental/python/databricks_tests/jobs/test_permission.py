import pytest

from databricks.bundles.jobs import Permission


def test_oneof_one():
    permission = Permission(
        level="CAN_VIEW",
        user_name="test@example.com",
    )

    assert permission


def test_oneof_none():
    with pytest.raises(ValueError) as exc_info:
        Permission(level="CAN_VIEW")  # FIXME should be enum

    assert exc_info.exconly() == (
        "ValueError: Permission must specify exactly one of 'user_name', "
        "'service_principal_name', 'group_name'"
    )


def test_oneof_both():
    with pytest.raises(ValueError) as exc_info:
        Permission(
            level="CAN_VIEW",  # FIXME should be enum
            user_name="test@example.com",
            service_principal_name="secret",
        )

    assert exc_info.exconly() == (
        "ValueError: Permission must specify exactly one of 'user_name', "
        "'service_principal_name', 'group_name'"
    )
