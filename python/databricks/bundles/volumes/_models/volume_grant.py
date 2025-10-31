from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList
from databricks.bundles.volumes._models.volume_grant_privilege import (
    VolumeGrantPrivilege,
    VolumeGrantPrivilegeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class VolumeGrant:
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[VolumeGrantPrivilege] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "VolumeGrantDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "VolumeGrantDict":
        return _transform_to_json_value(self)  # type:ignore


class VolumeGrantDict(TypedDict, total=False):
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[VolumeGrantPrivilegeParam]


VolumeGrantParam = VolumeGrantDict | VolumeGrant
