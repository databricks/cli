from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.jobs._models.deployment_spec import (
    DeploymentSpec,
    DeploymentSpecParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AiRuntimeTask:
    """
    :meta private: [EXPERIMENTAL]

    AiRuntimeTask: multi-node GPU compute task definition for Databricks AI
    Runtime workloads.

    Jobs-framework-level concepts (retries, per-task timeout, idempotency
    token, usage/budget policy, permissions) live on the surrounding
    TaskSettings / run-submit request and are intentionally NOT duplicated
    here. Users compose `ai_runtime_task` with the standard Jobs/DABs task
    wrapper to get those.
    """

    experiment: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] MLflow experiment name for this run. If an experiment with this name
    already exists under the calling user, the run is appended to it;
    otherwise a new experiment is created. To target a specific MLflow
    storage location (for example, when running as a service principal), set
    `mlflow_experiment_directory`.
    """

    code_source_path: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional workspace or UC volume path of the uploaded code-source
    archive. The CLI packages the user's local code directory into an
    archive and populates this. Customers calling the Jobs API directly
    should upload their archive to the workspace or a UC volume first and
    supply the resulting path here.
    
    When set, the training node exposes the value via the `$CODE_SOURCE`
    environment variable.
    """

    deployments: VariableOrList[DeploymentSpec] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Deployment specs for this task. Exactly one deployment is currently
    supported (a single entry where every node runs the same command); this
    is a current-Preview constraint. Role-split workloads (driver + worker,
    parameter server, separate eval node, etc.) with multiple entries are the
    eventual intent but not yet supported.
    """

    mlflow_experiment_directory: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional workspace directory under which the MLflow experiment named in
    `experiment` is created. Must start with `/Workspace`. Set this when
    running as a service principal that has no default user directory; for
    regular users the experiment defaults to the user's home directory.
    """

    mlflow_run: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional display name for the MLflow run created under `experiment`. If
    omitted, MLflow generates a default name.
    """

    @classmethod
    def from_dict(cls, value: "AiRuntimeTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AiRuntimeTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class AiRuntimeTaskDict(TypedDict, total=False):
    """"""

    experiment: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] MLflow experiment name for this run. If an experiment with this name
    already exists under the calling user, the run is appended to it;
    otherwise a new experiment is created. To target a specific MLflow
    storage location (for example, when running as a service principal), set
    `mlflow_experiment_directory`.
    """

    code_source_path: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional workspace or UC volume path of the uploaded code-source
    archive. The CLI packages the user's local code directory into an
    archive and populates this. Customers calling the Jobs API directly
    should upload their archive to the workspace or a UC volume first and
    supply the resulting path here.
    
    When set, the training node exposes the value via the `$CODE_SOURCE`
    environment variable.
    """

    deployments: VariableOrList[DeploymentSpecParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Deployment specs for this task. Exactly one deployment is currently
    supported (a single entry where every node runs the same command); this
    is a current-Preview constraint. Role-split workloads (driver + worker,
    parameter server, separate eval node, etc.) with multiple entries are the
    eventual intent but not yet supported.
    """

    mlflow_experiment_directory: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional workspace directory under which the MLflow experiment named in
    `experiment` is created. Must start with `/Workspace`. Set this when
    running as a service principal that has no default user directory; for
    regular users the experiment defaults to the user's home directory.
    """

    mlflow_run: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional display name for the MLflow run created under `experiment`. If
    omitted, MLflow generates a default name.
    """


AiRuntimeTaskParam = AiRuntimeTaskDict | AiRuntimeTask
