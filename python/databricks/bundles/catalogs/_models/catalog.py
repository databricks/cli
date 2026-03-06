from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.catalogs._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.catalogs._models.privilege_assignment import (
    PrivilegeAssignment,
    PrivilegeAssignmentParam,
)
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Catalog(Resource):
    """"""

    name: VariableOr[str]
    """
    Name of catalog.
    """

    comment: VariableOrOptional[str] = None
    """
    User-provided free-form text description.
    """

    connection_name: VariableOrOptional[str] = None
    """
    The name of the connection to an external data source.
    """

    grants: VariableOrList[PrivilegeAssignment] = field(default_factory=list)

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    options: VariableOrDict[str] = field(default_factory=dict)
    """
    A map of key-value properties attached to the securable.
    """

    properties: VariableOrDict[str] = field(default_factory=dict)
    """
    A map of key-value properties attached to the securable.
    """

    provider_name: VariableOrOptional[str] = None
    """
    The name of delta sharing provider.
    
    A Delta Sharing catalog is a catalog that is based on a Delta share on a remote sharing server.
    """

    share_name: VariableOrOptional[str] = None
    """
    The name of the share under the share provider.
    """

    storage_root: VariableOrOptional[str] = None
    """
    Storage root URL for managed tables within catalog.
    """

    @classmethod
    def from_dict(cls, value: "CatalogDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CatalogDict":
        return _transform_to_json_value(self)  # type:ignore


class CatalogDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]
    """
    Name of catalog.
    """

    comment: VariableOrOptional[str]
    """
    User-provided free-form text description.
    """

    connection_name: VariableOrOptional[str]
    """
    The name of the connection to an external data source.
    """

    grants: VariableOrList[PrivilegeAssignmentParam]

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    options: VariableOrDict[str]
    """
    A map of key-value properties attached to the securable.
    """

    properties: VariableOrDict[str]
    """
    A map of key-value properties attached to the securable.
    """

    provider_name: VariableOrOptional[str]
    """
    The name of delta sharing provider.
    
    A Delta Sharing catalog is a catalog that is based on a Delta share on a remote sharing server.
    """

    share_name: VariableOrOptional[str]
    """
    The name of the share under the share provider.
    """

    storage_root: VariableOrOptional[str]
    """
    Storage root URL for managed tables within catalog.
    """


CatalogParam = CatalogDict | Catalog
