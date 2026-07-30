from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DockerBasicAuth:
    """"""

    password: VariableOrOptional[str] = None
    """
    Password of the user
    """

    username: VariableOrOptional[str] = None
    """
    Name of the user
    """

    @classmethod
    def from_dict(cls, value: "DockerBasicAuthDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DockerBasicAuthDict":
        return _transform_to_json_value(self)  # type:ignore


class DockerBasicAuthDict(TypedDict, total=False):
    """"""

    password: VariableOrOptional[str]
    """
    Password of the user
    """

    username: VariableOrOptional[str]
    """
    Name of the user
    """


DockerBasicAuthParam = DockerBasicAuthDict | DockerBasicAuth
