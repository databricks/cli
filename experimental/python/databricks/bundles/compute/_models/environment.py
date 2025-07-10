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

    jar_dependencies: VariableOrList[str] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    List of jar dependencies, should be string representing volume paths. For example: `/Volumes/path/to/test.jar`.
    """

    @classmethod
    def from_dict(cls, value: "EnvironmentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "EnvironmentDict":
        return _transform_to_json_value(self)  # type:ignore


class EnvironmentDict(TypedDict, total=False):
    """"""

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

    jar_dependencies: VariableOrList[str]
    """
    :meta private: [EXPERIMENTAL]
    
    List of jar dependencies, should be string representing volume paths. For example: `/Volumes/path/to/test.jar`.
    """


EnvironmentParam = EnvironmentDict | Environment
