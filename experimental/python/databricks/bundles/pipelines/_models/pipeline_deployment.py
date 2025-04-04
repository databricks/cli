from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.deployment_kind import (
    DeploymentKind,
    DeploymentKindParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelineDeployment:
    """"""

    kind: VariableOrOptional[DeploymentKind] = None
    """
    The deployment method that manages the pipeline.
    """

    metadata_file_path: VariableOrOptional[str] = None
    """
    The path to the file containing metadata about the deployment.
    """

    @classmethod
    def from_dict(cls, value: "PipelineDeploymentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineDeploymentDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineDeploymentDict(TypedDict, total=False):
    """"""

    kind: VariableOrOptional[DeploymentKindParam]
    """
    The deployment method that manages the pipeline.
    """

    metadata_file_path: VariableOrOptional[str]
    """
    The path to the file containing metadata about the deployment.
    """


PipelineDeploymentParam = PipelineDeploymentDict | PipelineDeployment
