from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Environment:
    """
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and SDP's environment for classic and serverless pipelines.
    In this minimal environment spec, only pip and java dependencies are supported.
    """

    base_environment: VariableOrOptional[str] = None
    """
    The base environment this environment is built on top of. A base environment defines the environment version and a
    list of dependencies for serverless compute. The value can be a file path to a custom `env.yaml` file
    (e.g., `/Workspace/path/to/env.yaml`). Support for a Databricks-provided base environment ID
    (e.g., `workspace-base-environments/databricks_ai_v4`) and workspace base environment ID
    (e.g., `workspace-base-environments/dbe_b849b66e-b31a-4cb5-b161-1f2b10877fb7`) is in Beta.
    Either `environment_version` or `base_environment` can be provided.  For more information, see
    """

    client: VariableOrOptional[str] = None
    """
    [DEPRECATED] Use `environment_version` instead.
    """

    dependencies: VariableOrList[str] = field(default_factory=list)
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    """

    environment_version: VariableOrOptional[str] = None
    """
    Either `environment_version` or `base_environment` needs to be provided. Environment version used by the environment.
    Each version comes with a specific Python version and a set of Python packages.
    The version is a string, consisting of an integer.
    """

    java_dependencies: VariableOrList[str] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "EnvironmentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EnvironmentDict":
        return _transform_to_json_value(self)  # type:ignore


class EnvironmentDict(TypedDict, total=False):
    """"""

    base_environment: VariableOrOptional[str]
    """
    The base environment this environment is built on top of. A base environment defines the environment version and a
    list of dependencies for serverless compute. The value can be a file path to a custom `env.yaml` file
    (e.g., `/Workspace/path/to/env.yaml`). Support for a Databricks-provided base environment ID
    (e.g., `workspace-base-environments/databricks_ai_v4`) and workspace base environment ID
    (e.g., `workspace-base-environments/dbe_b849b66e-b31a-4cb5-b161-1f2b10877fb7`) is in Beta.
    Either `environment_version` or `base_environment` can be provided.  For more information, see
    """

    client: VariableOrOptional[str]
    """
    [DEPRECATED] Use `environment_version` instead.
    """

    dependencies: VariableOrList[str]
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    """

    environment_version: VariableOrOptional[str]
    """
    Either `environment_version` or `base_environment` needs to be provided. Environment version used by the environment.
    Each version comes with a specific Python version and a set of Python packages.
    The version is a string, consisting of an integer.
    """

    java_dependencies: VariableOrList[str]


EnvironmentParam = EnvironmentDict | Environment
