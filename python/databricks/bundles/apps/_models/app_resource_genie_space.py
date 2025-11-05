from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_genie_space_genie_space_permission import (
    AppResourceGenieSpaceGenieSpacePermission,
    AppResourceGenieSpaceGenieSpacePermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceGenieSpace:
    """"""

    name: VariableOr[str]

    permission: VariableOr[AppResourceGenieSpaceGenieSpacePermission]

    space_id: VariableOr[str]

    @classmethod
    def from_dict(cls, value: "AppResourceGenieSpaceDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceGenieSpaceDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceGenieSpaceDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]

    permission: VariableOr[AppResourceGenieSpaceGenieSpacePermissionParam]

    space_id: VariableOr[str]


AppResourceGenieSpaceParam = AppResourceGenieSpaceDict | AppResourceGenieSpace
