from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.docker_basic_auth import (
    DockerBasicAuth,
    DockerBasicAuthParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DockerImage:
    """"""

    basic_auth: VariableOrOptional[DockerBasicAuth] = None

    url: VariableOrOptional[str] = None
    """
    URL of the docker image.
    """

    @classmethod
    def from_dict(cls, value: "DockerImageDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DockerImageDict":
        return _transform_to_json_value(self)  # type:ignore


class DockerImageDict(TypedDict, total=False):
    """"""

    basic_auth: VariableOrOptional[DockerBasicAuthParam]

    url: VariableOrOptional[str]
    """
    URL of the docker image.
    """


DockerImageParam = DockerImageDict | DockerImage
