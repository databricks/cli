__all__ = [
    "Catalog",
    "CatalogDict",
    "CatalogGrant",
    "CatalogGrantDict",
    "CatalogGrantParam",
    "CatalogGrantPrivilege",
    "CatalogGrantPrivilegeParam",
    "CatalogParam",
    "Lifecycle",
    "LifecycleDict",
    "LifecycleParam",
    "Privilege",
    "PrivilegeAssignment",
    "PrivilegeAssignmentDict",
    "PrivilegeAssignmentParam",
    "PrivilegeParam",
]


from databricks.bundles.catalogs._models.catalog import (
    Catalog,
    CatalogDict,
    CatalogParam,
)
from databricks.bundles.catalogs._models.lifecycle import (
    Lifecycle,
    LifecycleDict,
    LifecycleParam,
)
from databricks.bundles.catalogs._models.privilege import Privilege, PrivilegeParam
from databricks.bundles.catalogs._models.privilege_assignment import (
    PrivilegeAssignment,
    PrivilegeAssignmentDict,
    PrivilegeAssignmentParam,
)

CatalogGrant = PrivilegeAssignment
CatalogGrantDict = PrivilegeAssignmentDict
CatalogGrantParam = PrivilegeAssignmentParam
CatalogGrantPrivilege = Privilege
CatalogGrantPrivilegeParam = PrivilegeParam
