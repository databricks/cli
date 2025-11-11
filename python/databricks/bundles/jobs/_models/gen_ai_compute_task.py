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
    """

    dl_runtime_image: VariableOr[str]
    """
    Runtime image
    """

    command: VariableOrOptional[str] = None
    """
    Command launcher to run the actual script, e.g. bash, python etc.
    """

    compute: VariableOrOptional[ComputeConfig] = None

    mlflow_experiment_name: VariableOrOptional[str] = None
    """
    Optional string containing the name of the MLflow experiment to log the run to. If name is not
    found, backend will create the mlflow experiment using the name.
    """

    source: VariableOrOptional[Source] = None
    """
    Optional location type of the training script. When set to `WORKSPACE`, the script will be retrieved from the local Databricks workspace. When set to `GIT`, the script will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Script is located in Databricks workspace.
    * `GIT`: Script is located in cloud Git provider.
    """

    training_script_path: VariableOrOptional[str] = None
    """
    The training script file path to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    yaml_parameters: VariableOrOptional[str] = None
    """
    Optional string containing model parameters passed to the training script in yaml format.
    If present, then the content in yaml_parameters_file_path will be ignored.
    """

    yaml_parameters_file_path: VariableOrOptional[str] = None
    """
    Optional path to a YAML file containing model parameters passed to the training script.
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
    Runtime image
    """

    command: VariableOrOptional[str]
    """
    Command launcher to run the actual script, e.g. bash, python etc.
    """

    compute: VariableOrOptional[ComputeConfigParam]

    mlflow_experiment_name: VariableOrOptional[str]
    """
    Optional string containing the name of the MLflow experiment to log the run to. If name is not
    found, backend will create the mlflow experiment using the name.
    """

    source: VariableOrOptional[SourceParam]
    """
    Optional location type of the training script. When set to `WORKSPACE`, the script will be retrieved from the local Databricks workspace. When set to `GIT`, the script will be retrieved from a Git repository
    defined in `git_source`. If the value is empty, the task will use `GIT` if `git_source` is defined and `WORKSPACE` otherwise.
    * `WORKSPACE`: Script is located in Databricks workspace.
    * `GIT`: Script is located in cloud Git provider.
    """

    training_script_path: VariableOrOptional[str]
    """
    The training script file path to be executed. Cloud file URIs (such as dbfs:/, s3:/, adls:/, gcs:/) and workspace paths are supported. For python files stored in the Databricks workspace, the path must be absolute and begin with `/`. For files stored in a remote repository, the path must be relative. This field is required.
    """

    yaml_parameters: VariableOrOptional[str]
    """
    Optional string containing model parameters passed to the training script in yaml format.
    If present, then the content in yaml_parameters_file_path will be ignored.
    """

    yaml_parameters_file_path: VariableOrOptional[str]
    """
    Optional path to a YAML file containing model parameters passed to the training script.
    """


GenAiComputeTaskParam = GenAiComputeTaskDict | GenAiComputeTask
