"""
Backward compatibility: old per-resource grant types must remain importable as aliases
to the unified PrivilegeAssignment / Privilege types.
"""

from databricks.bundles.catalogs import (
    CatalogGrant,
    CatalogGrantDict,
    CatalogGrantParam,
    CatalogGrantPrivilege,
    CatalogGrantPrivilegeParam,
    Privilege,
    PrivilegeAssignment,
    PrivilegeParam,
)
from databricks.bundles.schemas import (
    Privilege as SchemaPrivilege,
)
from databricks.bundles.schemas import (
    PrivilegeAssignment as SchemaPrivilegeAssignment,
)
from databricks.bundles.schemas import (
    SchemaGrant,
    SchemaGrantDict,
    SchemaGrantParam,
    SchemaGrantPrivilege,
    SchemaGrantPrivilegeParam,
)
from databricks.bundles.volumes import (
    Privilege as VolumePrivilege,
)
from databricks.bundles.volumes import (
    PrivilegeAssignment as VolumePrivilegeAssignment,
)
from databricks.bundles.volumes import (
    VolumeGrant,
    VolumeGrantDict,
    VolumeGrantParam,
    VolumeGrantPrivilege,
    VolumeGrantPrivilegeParam,
)


def test_catalog_grant_aliases():
    assert CatalogGrant is PrivilegeAssignment
    assert CatalogGrantPrivilege is Privilege
    # Param types are union aliases; verify they resolve without error
    assert CatalogGrantDict is not None
    assert CatalogGrantParam is not None
    assert CatalogGrantPrivilegeParam is PrivilegeParam


def test_schema_grant_aliases():
    assert SchemaGrant is SchemaPrivilegeAssignment
    assert SchemaGrantPrivilege is SchemaPrivilege
    assert SchemaGrantDict is not None
    assert SchemaGrantParam is not None
    assert SchemaGrantPrivilegeParam is not None


def test_volume_grant_aliases():
    assert VolumeGrant is VolumePrivilegeAssignment
    assert VolumeGrantPrivilege is VolumePrivilege
    assert VolumeGrantDict is not None
    assert VolumeGrantParam is not None
    assert VolumeGrantPrivilegeParam is not None


def test_catalog_grant_is_usable():
    g = CatalogGrant(
        principal="user@example.com", privileges=[CatalogGrantPrivilege.SELECT]
    )
    assert g.principal == "user@example.com"
    assert g.privileges == [Privilege.SELECT]


def test_schema_grant_is_usable():
    g = SchemaGrant(principal="group", privileges=[SchemaGrantPrivilege.USE_SCHEMA])
    assert g.principal == "group"
    assert g.privileges == [SchemaPrivilege.USE_SCHEMA]


def test_volume_grant_is_usable():
    g = VolumeGrant(principal="svc", privileges=[VolumeGrantPrivilege.READ_VOLUME])
    assert g.principal == "svc"
    assert g.privileges == [VolumePrivilege.READ_VOLUME]


def test_catalog_grant_dict_roundtrip():
    grant: CatalogGrantDict = {
        "principal": "user@example.com",
        "privileges": ["SELECT"],
    }
    obj = CatalogGrant.from_dict(grant)
    assert obj.principal == "user@example.com"
    assert obj.privileges == [Privilege.SELECT]


def test_schema_grant_dict_roundtrip():
    grant: SchemaGrantDict = {"principal": "group", "privileges": ["USE_SCHEMA"]}
    obj = SchemaGrant.from_dict(grant)
    assert obj.privileges == [SchemaPrivilege.USE_SCHEMA]


def test_volume_grant_dict_roundtrip():
    grant: VolumeGrantDict = {"principal": "svc", "privileges": ["READ_VOLUME"]}
    obj = VolumeGrant.from_dict(grant)
    assert obj.privileges == [VolumePrivilege.READ_VOLUME]
