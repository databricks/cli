from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Job(Resource):
    """"""

    description: VariableOrOptional[str] = None
    """
    An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
    """

    name: VariableOrOptional[str] = None
    """
    An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
    """

    @classmethod
    def from_dict(cls, value: "JobDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobDict":
        return _transform_to_json_value(self)  # type:ignore


class JobDict(TypedDict, total=False):
    """"""

    description: VariableOrOptional[str]
    """
    An optional description for the job. The maximum length is 27700 characters in UTF-8 encoding.
    """

    name: VariableOrOptional[str]
    """
    An optional name for the job. The maximum length is 4096 bytes in UTF-8 encoding.
    """


JobParam = JobDict | Job
