from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Environment:
    """
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
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

    environment_version: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    We renamed `client` to `environment_version` in notebook exports. This field is meant solely so that imported notebooks with `environment_version` can be deserialized
    correctly, in a backwards-compatible way (i.e. if `client` is specified instead of `environment_version`, it will be deserialized correctly). Do NOT use this field
    for any other purpose, e.g. notebook storage.
    This field is not yet exposed to customers (e.g. in the jobs API).
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

    environment_version: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    We renamed `client` to `environment_version` in notebook exports. This field is meant solely so that imported notebooks with `environment_version` can be deserialized
    correctly, in a backwards-compatible way (i.e. if `client` is specified instead of `environment_version`, it will be deserialized correctly). Do NOT use this field
    for any other purpose, e.g. notebook storage.
    This field is not yet exposed to customers (e.g. in the jobs API).
    """

    jar_dependencies: VariableOrList[str]
    """
    :meta private: [EXPERIMENTAL]
    
    List of jar dependencies, should be string representing volume paths. For example: `/Volumes/path/to/test.jar`.
    """


EnvironmentParam = EnvironmentDict | Environment
