from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.schemas._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.schemas._models.schema_grant import (
    SchemaGrant,
    SchemaGrantParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Schema(Resource):
    """"""

    catalog_name: VariableOr[str]
    """
    Name of parent catalog.
    """

    name: VariableOr[str]
    """
    Name of schema, relative to parent catalog.
    """

    comment: VariableOrOptional[str] = None
    """
    User-provided free-form text description.
    """

    grants: VariableOrList[SchemaGrant] = field(default_factory=list)

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    properties: VariableOrDict[str] = field(default_factory=dict)

    storage_root: VariableOrOptional[str] = None
    """
    Storage root URL for managed tables within schema.
    """

    @classmethod
    def from_dict(cls, value: "SchemaDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SchemaDict":
        return _transform_to_json_value(self)  # type:ignore


class SchemaDict(TypedDict, total=False):
    """"""

    catalog_name: VariableOr[str]
    """
    Name of parent catalog.
    """

    name: VariableOr[str]
    """
    Name of schema, relative to parent catalog.
    """

    comment: VariableOrOptional[str]
    """
    User-provided free-form text description.
    """

    grants: VariableOrList[SchemaGrantParam]

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    properties: VariableOrDict[str]

    storage_root: VariableOrOptional[str]
    """
    Storage root URL for managed tables within schema.
    """


SchemaParam = SchemaDict | Schema
