from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrDict

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class RunJobTask:
    """"""

    job_id: VariableOr[int]
    """
    ID of the job to trigger.
    """

    job_parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    Job-level parameters used to trigger the job.
    """

    @classmethod
    def from_dict(cls, value: "RunJobTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RunJobTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class RunJobTaskDict(TypedDict, total=False):
    """"""

    job_id: VariableOr[int]
    """
    ID of the job to trigger.
    """

    job_parameters: VariableOrDict[str]
    """
    Job-level parameters used to trigger the job.
    """


RunJobTaskParam = RunJobTaskDict | RunJobTask
