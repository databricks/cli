from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Environment:
    """
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
    (Note: DLT uses a copied version of the Environment proto below, at //spark/pipelines/api/protos/copied/libraries-environments-copy.proto)
    In this minimal environment spec, only pip dependencies are supported.
    """

    client: VariableOr[str]
    """
    Client version used by the environment
    The client is the user-facing environment of the runtime.
    Each client comes with a specific set of pre-installed libraries.
    The version is a string, consisting of the major client version.
    """

    dependencies: VariableOrList[str] = field(default_factory=list)
    """
    List of pip dependencies, as supported by the version of pip in this environment.
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

    client: VariableOr[str]
    """
    Client version used by the environment
    The client is the user-facing environment of the runtime.
    Each client comes with a specific set of pre-installed libraries.
    The version is a string, consisting of the major client version.
    """

    dependencies: VariableOrList[str]
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    """

    jar_dependencies: VariableOrList[str]
    """
    :meta private: [EXPERIMENTAL]
    
    List of jar dependencies, should be string representing volume paths. For example: `/Volumes/path/to/test.jar`.
    """


EnvironmentParam = EnvironmentDict | Environment
