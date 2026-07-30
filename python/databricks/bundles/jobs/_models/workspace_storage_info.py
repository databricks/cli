from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class WorkspaceStorageInfo:
    """
    A storage location in Workspace Filesystem (WSFS)
    """

    destination: VariableOr[str]
    """
    wsfs destination, e.g. `workspace:/cluster-init-scripts/setup-datadog.sh`
    """

    @classmethod
    def from_dict(cls, value: "WorkspaceStorageInfoDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "WorkspaceStorageInfoDict":
        return _transform_to_json_value(self)  # type:ignore


class WorkspaceStorageInfoDict(TypedDict, total=False):
    """"""

    destination: VariableOr[str]
    """
    wsfs destination, e.g. `workspace:/cluster-init-scripts/setup-datadog.sh`
    """


WorkspaceStorageInfoParam = WorkspaceStorageInfoDict | WorkspaceStorageInfo
