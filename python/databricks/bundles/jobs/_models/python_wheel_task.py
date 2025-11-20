from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrDict, VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PythonWheelTask:
    """"""

    entry_point: VariableOr[str]
    """
    Named entry point to use, if it does not exist in the metadata of the package it executes the function from the package directly using `$packageName.$entryPoint()`
    """

    package_name: VariableOr[str]
    """
    Name of the package to execute
    """

    named_parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    Command-line parameters passed to Python wheel task in the form of `["--name=task", "--data=dbfs:/path/to/data.json"]`. Leave it empty if `parameters` is not null.
    """

    parameters: VariableOrList[str] = field(default_factory=list)
    """
    Command-line parameters passed to Python wheel task. Leave it empty if `named_parameters` is not null.
    """

    @classmethod
    def from_dict(cls, value: "PythonWheelTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PythonWheelTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class PythonWheelTaskDict(TypedDict, total=False):
    """"""

    entry_point: VariableOr[str]
    """
    Named entry point to use, if it does not exist in the metadata of the package it executes the function from the package directly using `$packageName.$entryPoint()`
    """

    package_name: VariableOr[str]
    """
    Name of the package to execute
    """

    named_parameters: VariableOrDict[str]
    """
    Command-line parameters passed to Python wheel task in the form of `["--name=task", "--data=dbfs:/path/to/data.json"]`. Leave it empty if `parameters` is not null.
    """

    parameters: VariableOrList[str]
    """
    Command-line parameters passed to Python wheel task. Leave it empty if `named_parameters` is not null.
    """


PythonWheelTaskParam = PythonWheelTaskDict | PythonWheelTask
