from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_deployment_state import (
    AppDeploymentState,
    AppDeploymentStateParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppDeploymentStatus:
    """"""

    message: VariableOrOptional[str] = None

    state: VariableOrOptional[AppDeploymentState] = None

    @classmethod
    def from_dict(cls, value: "AppDeploymentStatusDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppDeploymentStatusDict":
        return _transform_to_json_value(self)  # type:ignore


class AppDeploymentStatusDict(TypedDict, total=False):
    """"""

    message: VariableOrOptional[str]

    state: VariableOrOptional[AppDeploymentStateParam]


AppDeploymentStatusParam = AppDeploymentStatusDict | AppDeploymentStatus
