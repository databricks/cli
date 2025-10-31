from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PipelinesEnvironment:
    """
    The environment entity used to preserve serverless environment side panel, jobs' environment for non-notebook task, and DLT's environment for classic and serverless pipelines.
    In this minimal environment spec, only pip dependencies are supported.
    """

    dependencies: VariableOrList[str] = field(default_factory=list)
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    Each dependency is a pip requirement file line https://pip.pypa.io/en/stable/reference/requirements-file-format/
    Allowed dependency could be <requirement specifier>, <archive url/path>, <local project path>(WSFS or Volumes in Databricks), <vcs project url>
    """

    @classmethod
    def from_dict(cls, value: "PipelinesEnvironmentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelinesEnvironmentDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelinesEnvironmentDict(TypedDict, total=False):
    """"""

    dependencies: VariableOrList[str]
    """
    List of pip dependencies, as supported by the version of pip in this environment.
    Each dependency is a pip requirement file line https://pip.pypa.io/en/stable/reference/requirements-file-format/
    Allowed dependency could be <requirement specifier>, <archive url/path>, <local project path>(WSFS or Volumes in Databricks), <vcs project url>
    """


PipelinesEnvironmentParam = PipelinesEnvironmentDict | PipelinesEnvironment
