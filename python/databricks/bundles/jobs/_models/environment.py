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
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
    In this minimal environment spec, only pip dependencies are supported.
    """

    base_environment: VariableOrOptional[str] = None
    """
    The `base_environment` key refers to an `env.yaml` file that specifies an environment version and a collection of dependencies required for the environment setup.
    This `env.yaml` file may itself include a `base_environment` reference pointing to another `env_1.yaml` file. However, when used as a base environment, `env_1.yaml` (or further nested references) will not be processed or included in the final environment, meaning that the resolution of `base_environment` references is not recursive.
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
    Required. Environment version used by the environment.
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
    The `base_environment` key refers to an `env.yaml` file that specifies an environment version and a collection of dependencies required for the environment setup.
    This `env.yaml` file may itself include a `base_environment` reference pointing to another `env_1.yaml` file. However, when used as a base environment, `env_1.yaml` (or further nested references) will not be processed or included in the final environment, meaning that the resolution of `base_environment` references is not recursive.
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
    Required. Environment version used by the environment.
    Each version comes with a specific Python version and a set of Python packages.
    The version is a string, consisting of an integer.
    """

    java_dependencies: VariableOrList[str]


EnvironmentParam = EnvironmentDict | Environment
