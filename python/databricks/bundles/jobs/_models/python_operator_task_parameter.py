from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PythonOperatorTaskParameter:
    """
    :meta private: [EXPERIMENTAL]
    """

    name: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    value: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    @classmethod
    def from_dict(cls, value: "PythonOperatorTaskParameterDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PythonOperatorTaskParameterDict":
        return _transform_to_json_value(self)  # type:ignore


class PythonOperatorTaskParameterDict(TypedDict, total=False):
    """"""

    name: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    value: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """


PythonOperatorTaskParameterParam = PythonOperatorTaskParameterDict | PythonOperatorTaskParameter
