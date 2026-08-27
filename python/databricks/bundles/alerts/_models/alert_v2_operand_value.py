from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertV2OperandValue:
    """"""

    bool_value: VariableOrOptional[bool] = None

    double_value: VariableOrOptional[float] = None

    string_value: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "AlertV2OperandValueDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertV2OperandValueDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertV2OperandValueDict(TypedDict, total=False):
    """"""

    bool_value: VariableOrOptional[bool]

    double_value: VariableOrOptional[float]

    string_value: VariableOrOptional[str]


AlertV2OperandValueParam = AlertV2OperandValueDict | AlertV2OperandValue
