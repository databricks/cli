from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_job_job_permission import (
    AppResourceJobJobPermission,
    AppResourceJobJobPermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceJob:
    """"""

    id: VariableOr[str]

    permission: VariableOr[AppResourceJobJobPermission]

    @classmethod
    def from_dict(cls, value: "AppResourceJobDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceJobDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceJobDict(TypedDict, total=False):
    """"""

    id: VariableOr[str]

    permission: VariableOr[AppResourceJobJobPermissionParam]


AppResourceJobParam = AppResourceJobDict | AppResourceJob
