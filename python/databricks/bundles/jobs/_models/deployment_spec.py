from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.compute_spec import (
    ComputeSpec,
    ComputeSpecParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DeploymentSpec:
    """
    :meta private: [EXPERIMENTAL]

    DeploymentSpec: configuration for one deployment within an AiRuntimeTask.
    Each entry in `AiRuntimeTask.deployments` describes a group of nodes that
    share the same command and compute. Many single-program training
    algorithms use a single entry where every node runs the same command;
    role-split workloads (driver + worker, parameter server, separate eval
    node, etc.) use multiple entries.
    """

    command_path: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Workspace path of the bash script to execute on each node in this
    deployment. The CLI uploads the user's script and populates this.
    Customers calling the Jobs API directly should upload their script to
    the workspace first and supply the resulting path here.
    """

    compute: VariableOr[ComputeSpec]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Compute resources allocated to each node in this deployment.
    """

    name: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional human-readable name for this deployment (for example, `driver`,
    `worker`, `param_server`). Used for log and UI display. Distinct names
    are recommended so deployments can be told apart, but uniqueness is not
    enforced.
    """

    @classmethod
    def from_dict(cls, value: "DeploymentSpecDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DeploymentSpecDict":
        return _transform_to_json_value(self)  # type:ignore


class DeploymentSpecDict(TypedDict, total=False):
    """"""

    command_path: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Workspace path of the bash script to execute on each node in this
    deployment. The CLI uploads the user's script and populates this.
    Customers calling the Jobs API directly should upload their script to
    the workspace first and supply the resulting path here.
    """

    compute: VariableOr[ComputeSpecParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Compute resources allocated to each node in this deployment.
    """

    name: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional human-readable name for this deployment (for example, `driver`,
    `worker`, `param_server`). Used for log and UI display. Distinct names
    are recommended so deployments can be told apart, but uniqueness is not
    enforced.
    """


DeploymentSpecParam = DeploymentSpecDict | DeploymentSpec
