from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.jobs._models.python_operator_task_parameter import (
    PythonOperatorTaskParameter,
    PythonOperatorTaskParameterParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PythonOperatorTask:
    """
    :meta private: [EXPERIMENTAL]
    """

    main: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Fully qualified name of the main class or function.
    For example, `my_project.my_function` or `my_project.MyOperator`.
    """

    parameters: VariableOrList[PythonOperatorTaskParameter] = field(
        default_factory=list
    )
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] An ordered list of task parameters.
    TODO(JOBS-30885): Add limits for parameters.
    """

    @classmethod
    def from_dict(cls, value: "PythonOperatorTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PythonOperatorTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class PythonOperatorTaskDict(TypedDict, total=False):
    """"""

    main: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Fully qualified name of the main class or function.
    For example, `my_project.my_function` or `my_project.MyOperator`.
    """

    parameters: VariableOrList[PythonOperatorTaskParameterParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] An ordered list of task parameters.
    TODO(JOBS-30885): Add limits for parameters.
    """


PythonOperatorTaskParam = PythonOperatorTaskDict | PythonOperatorTask
