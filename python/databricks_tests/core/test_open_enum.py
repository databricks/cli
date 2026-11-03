"""Privilege is generated as an open enum: unknown values are accepted instead of raising."""

import pytest

from databricks.bundles.schemas import Privilege, PrivilegeAssignment


def test_known_value():
    assert Privilege("SELECT") is Privilege.SELECT


def test_unknown_value_is_accepted():
    privilege = Privilege("CREATE_SECRET")

    assert isinstance(privilege, Privilege)
    assert privilege.value == "CREATE_SECRET"
    assert privilege.name == "CREATE_SECRET"


def test_unknown_value_is_cached():
    assert Privilege("CREATE_FEATURE") is Privilege("CREATE_FEATURE")


def test_unknown_non_string_raises():
    with pytest.raises(ValueError):
        Privilege(42)


def test_unknown_value_round_trip():
    assignment = PrivilegeAssignment.from_dict(
        {"principal": "user@databricks.com", "privileges": ["CREATE_SECRET", "SELECT"]}
    )

    assert assignment == PrivilegeAssignment(
        principal="user@databricks.com",
        privileges=[Privilege("CREATE_SECRET"), Privilege.SELECT],
    )
    assert assignment.as_dict() == {
        "principal": "user@databricks.com",
        "privileges": ["CREATE_SECRET", "SELECT"],
    }
