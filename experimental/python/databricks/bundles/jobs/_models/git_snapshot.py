from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GitSnapshot:
    """
    Read-only state of the remote repository at the time the job was run. This field is only included on job runs.
    """

    used_commit: VariableOrOptional[str] = None
    """
    Commit that was used to execute the run. If git_branch was specified, this points to the HEAD of the branch at the time of the run; if git_tag was specified, this points to the commit the tag points to.
    """

    @classmethod
    def from_dict(cls, value: "GitSnapshotDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GitSnapshotDict":
        return _transform_to_json_value(self)  # type:ignore


class GitSnapshotDict(TypedDict, total=False):
    """"""

    used_commit: VariableOrOptional[str]
    """
    Commit that was used to execute the run. If git_branch was specified, this points to the HEAD of the branch at the time of the run; if git_tag was specified, this points to the commit the tag points to.
    """


GitSnapshotParam = GitSnapshotDict | GitSnapshot
