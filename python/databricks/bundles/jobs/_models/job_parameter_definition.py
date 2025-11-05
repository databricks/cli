from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobParameterDefinition:
    """"""

    default: VariableOr[str]
    """
    Default value of the parameter.
    """

    name: VariableOr[str]
    """
    The name of the defined parameter. May only contain alphanumeric characters, `_`, `-`, and `.`
    """

    @classmethod
    def from_dict(cls, value: "JobParameterDefinitionDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobParameterDefinitionDict":
        return _transform_to_json_value(self)  # type:ignore


class JobParameterDefinitionDict(TypedDict, total=False):
    """"""

    default: VariableOr[str]
    """
    Default value of the parameter.
    """

    name: VariableOr[str]
    """
    The name of the defined parameter. May only contain alphanumeric characters, `_`, `-`, and `.`
    """


JobParameterDefinitionParam = JobParameterDefinitionDict | JobParameterDefinition
