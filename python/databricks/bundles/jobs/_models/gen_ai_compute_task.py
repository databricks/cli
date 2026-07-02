from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional
from databricks.bundles.jobs._models.compute_config import (
    ComputeConfig,
    ComputeConfigParam,
)
from databricks.bundles.jobs._models.source import Source, SourceParam

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GenAiComputeTask:
    """
    :meta private: [EXPERIMENTAL]

    DEPRECATED — use `AiRuntimeTask` for all new BYOT multi-node GPU
    workloads (see ai_runtime_task.proto). `AiRuntimeTask` is the only
    supported BYOT task type for new workloads; this proto is retained only
    for AIR CLI (fka SGCLI) pywheel backwards compatibility and will be
    removed once the pywheel → databricks-cli migration completes (post-
    PuPr).
    """

    dl_runtime_image: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Runtime image
    """

    command: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Command launcher to run the actual script, e.g. bash, python etc.
    """

    compute: VariableOrOptional[ComputeConfig] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    mlflow_experiment_name: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional string containing the name of the MLflow experiment to log the run to. If name is not
    found, backend will create the mlflow experiment using the name.
    """

    source: VariableOrOptional[Source] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional location type of the training script. When set to `WORKSPACE`, the script will be retrieved from the local Databricks workspace. When set to `GIT`, the script will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Script is located in Databricks workspace.
    * `GIT`: Script is located in cloud Git provider.
    """

    training_script_path: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The training script file path to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    yaml_parameters: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional string containing model parameters passed to the training script in yaml format.
    If present, then the content in yaml_parameters_file_path will be ignored.
    """

    yaml_parameters_file_path: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional path to a YAML file containing model parameters passed to the training script.
    """

    @classmethod
    def from_dict(cls, value: "GenAiComputeTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GenAiComputeTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class GenAiComputeTaskDict(TypedDict, total=False):
    """"""

    dl_runtime_image: VariableOr[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Runtime image
    """

    command: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Command launcher to run the actual script, e.g. bash, python etc.
    """

    compute: VariableOrOptional[ComputeConfigParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    mlflow_experiment_name: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional string containing the name of the MLflow experiment to log the run to. If name is not
    found, backend will create the mlflow experiment using the name.
    """

    source: VariableOrOptional[SourceParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional location type of the training script. When set to `WORKSPACE`, the script will be retrieved from the local Databricks workspace. When set to `GIT`, the script will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Script is located in Databricks workspace.
    * `GIT`: Script is located in cloud Git provider.
    """

    training_script_path: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] The training script file path to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    yaml_parameters: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional string containing model parameters passed to the training script in yaml format.
    If present, then the content in yaml_parameters_file_path will be ignored.
    """

    yaml_parameters_file_path: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional path to a YAML file containing model parameters passed to the training script.
    """


GenAiComputeTaskParam = GenAiComputeTaskDict | GenAiComputeTask
