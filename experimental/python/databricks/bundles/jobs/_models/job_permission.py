from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.job_permission_level import (
    JobPermissionLevel,
    JobPermissionLevelParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobPermission:
    """"""

    level: VariableOr[JobPermissionLevel]

    group_name: VariableOrOptional[str] = None

    service_principal_name: VariableOrOptional[str] = None

    user_name: VariableOrOptional[str] = None

    def __post_init__(self):
        union_fields = [
            self.user_name,
            self.service_principal_name,
            self.group_name,
        ]

        if sum(f is not None for f in union_fields) != 1:
            raise ValueError(
                "JobPermission must specify exactly one of 'user_name', 'service_principal_name', 'group_name'"
            )

    @classmethod
    def from_dict(cls, value: "JobPermissionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobPermissionDict":
        return _transform_to_json_value(self)  # type:ignore


class JobPermissionDict(TypedDict, total=False):
    """"""

    level: VariableOr[JobPermissionLevelParam]

    group_name: VariableOrOptional[str]

    service_principal_name: VariableOrOptional[str]

    user_name: VariableOrOptional[str]


JobPermissionParam = JobPermissionDict | JobPermission
