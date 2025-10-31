from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppDeploymentArtifacts:
    """"""

    source_code_path: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "AppDeploymentArtifactsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppDeploymentArtifactsDict":
        return _transform_to_json_value(self)  # type:ignore


class AppDeploymentArtifactsDict(TypedDict, total=False):
    """"""

    source_code_path: VariableOrOptional[str]


AppDeploymentArtifactsParam = AppDeploymentArtifactsDict | AppDeploymentArtifacts
