from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class PythonPyPiLibrary:
    """"""

    package: VariableOr[str]
    """
    The name of the pypi package to install. An optional exact version specification is also
    supported. Examples: "simplejson" and "simplejson==3.8.0".
    """

    repo: VariableOrOptional[str] = None
    """
    The repository where the package can be found. If not specified, the default pip index is
    used.
    """

    @classmethod
    def from_dict(cls, value: "PythonPyPiLibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PythonPyPiLibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class PythonPyPiLibraryDict(TypedDict, total=False):
    """"""

    package: VariableOr[str]
    """
    The name of the pypi package to install. An optional exact version specification is also
    supported. Examples: "simplejson" and "simplejson==3.8.0".
    """

    repo: VariableOrOptional[str]
    """
    The repository where the package can be found. If not specified, the default pip index is
    used.
    """


PythonPyPiLibraryParam = PythonPyPiLibraryDict | PythonPyPiLibrary
