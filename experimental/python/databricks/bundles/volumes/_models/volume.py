from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.volumes._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.volumes._models.volume_grant import (
    VolumeGrant,
    VolumeGrantParam,
)
from databricks.bundles.volumes._models.volume_type import VolumeType, VolumeTypeParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Volume(Resource):
    """"""

    catalog_name: VariableOr[str]
    """
    The name of the catalog where the schema and the volume are
    """

    name: VariableOr[str]
    """
    The name of the volume
    """

    schema_name: VariableOr[str]
    """
    The name of the schema where the volume is
    """

    comment: VariableOrOptional[str] = None
    """
    The comment attached to the volume
    """

    grants: VariableOrList[VolumeGrant] = field(default_factory=list)

    lifecycle: VariableOrOptional[Lifecycle] = None
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    storage_location: VariableOrOptional[str] = None
    """
    The storage location on the cloud
    """

    volume_type: VariableOrOptional[VolumeType] = None

    @classmethod
    def from_dict(cls, value: "VolumeDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "VolumeDict":
        return _transform_to_json_value(self)  # type:ignore


class VolumeDict(TypedDict, total=False):
    """"""

    catalog_name: VariableOr[str]
    """
    The name of the catalog where the schema and the volume are
    """

    name: VariableOr[str]
    """
    The name of the volume
    """

    schema_name: VariableOr[str]
    """
    The name of the schema where the volume is
    """

    comment: VariableOrOptional[str]
    """
    The comment attached to the volume
    """

    grants: VariableOrList[VolumeGrantParam]

    lifecycle: VariableOrOptional[LifecycleParam]
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    storage_location: VariableOrOptional[str]
    """
    The storage location on the cloud
    """

    volume_type: VariableOrOptional[VolumeTypeParam]


VolumeParam = VolumeDict | Volume
