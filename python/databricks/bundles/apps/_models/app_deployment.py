from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_deployment_artifacts import (
    AppDeploymentArtifacts,
    AppDeploymentArtifactsParam,
)
from databricks.bundles.apps._models.app_deployment_mode import (
    AppDeploymentMode,
    AppDeploymentModeParam,
)
from databricks.bundles.apps._models.app_deployment_status import (
    AppDeploymentStatus,
    AppDeploymentStatusParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppDeployment:
    """"""

    create_time: VariableOrOptional[str] = None

    creator: VariableOrOptional[str] = None

    deployment_artifacts: VariableOrOptional[AppDeploymentArtifacts] = None

    deployment_id: VariableOrOptional[str] = None

    mode: VariableOrOptional[AppDeploymentMode] = None

    source_code_path: VariableOrOptional[str] = None

    status: VariableOrOptional[AppDeploymentStatus] = None

    update_time: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "AppDeploymentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppDeploymentDict":
        return _transform_to_json_value(self)  # type:ignore


class AppDeploymentDict(TypedDict, total=False):
    """"""

    create_time: VariableOrOptional[str]

    creator: VariableOrOptional[str]

    deployment_artifacts: VariableOrOptional[AppDeploymentArtifactsParam]

    deployment_id: VariableOrOptional[str]

    mode: VariableOrOptional[AppDeploymentModeParam]

    source_code_path: VariableOrOptional[str]

    status: VariableOrOptional[AppDeploymentStatusParam]

    update_time: VariableOrOptional[str]


AppDeploymentParam = AppDeploymentDict | AppDeployment
