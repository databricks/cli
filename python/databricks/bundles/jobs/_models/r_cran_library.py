from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class RCranLibrary:
    """"""

    package: VariableOr[str]
    """
    The name of the CRAN package to install.
    """

    repo: VariableOrOptional[str] = None
    """
    The repository where the package can be found. If not specified, the default CRAN repo is used.
    """

    @classmethod
    def from_dict(cls, value: "RCranLibraryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RCranLibraryDict":
        return _transform_to_json_value(self)  # type:ignore


class RCranLibraryDict(TypedDict, total=False):
    """"""

    package: VariableOr[str]
    """
    The name of the CRAN package to install.
    """

    repo: VariableOrOptional[str]
    """
    The repository where the package can be found. If not specified, the default CRAN repo is used.
    """


RCranLibraryParam = RCranLibraryDict | RCranLibrary
