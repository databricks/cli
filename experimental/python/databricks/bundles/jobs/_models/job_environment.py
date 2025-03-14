from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.environment import Environment, EnvironmentParam
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobEnvironment:
    """"""

    environment_key: VariableOr[str]
    """
    The key of an environment. It has to be unique within a job.
    """

    spec: VariableOrOptional[Environment] = None

    @classmethod
    def from_dict(cls, value: "JobEnvironmentDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobEnvironmentDict":
        return _transform_to_json_value(self)  # type:ignore


class JobEnvironmentDict(TypedDict, total=False):
    """"""

    environment_key: VariableOr[str]
    """
    The key of an environment. It has to be unique within a job.
    """

    spec: VariableOrOptional[EnvironmentParam]


JobEnvironmentParam = JobEnvironmentDict | JobEnvironment
