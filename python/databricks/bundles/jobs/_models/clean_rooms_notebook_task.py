from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CleanRoomsNotebookTask:
    """"""

    clean_room_name: VariableOr[str]
    """
    The clean room that the notebook belongs to.
    """

    notebook_name: VariableOr[str]
    """
    Name of the notebook being run.
    """

    etag: VariableOrOptional[str] = None
    """
    Checksum to validate the freshness of the notebook resource (i.e. the notebook being run is the latest version).
    It can be fetched by calling the :method:cleanroomassets/get API.
    """

    notebook_base_parameters: VariableOrDict[str] = field(default_factory=dict)
    """
    Base parameters to be used for the clean room notebook job.
    """

    @classmethod
    def from_dict(cls, value: "CleanRoomsNotebookTaskDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CleanRoomsNotebookTaskDict":
        return _transform_to_json_value(self)  # type:ignore


class CleanRoomsNotebookTaskDict(TypedDict, total=False):
    """"""

    clean_room_name: VariableOr[str]
    """
    The clean room that the notebook belongs to.
    """

    notebook_name: VariableOr[str]
    """
    Name of the notebook being run.
    """

    etag: VariableOrOptional[str]
    """
    Checksum to validate the freshness of the notebook resource (i.e. the notebook being run is the latest version).
    It can be fetched by calling the :method:cleanroomassets/get API.
    """

    notebook_base_parameters: VariableOrDict[str]
    """
    Base parameters to be used for the clean room notebook job.
    """


CleanRoomsNotebookTaskParam = CleanRoomsNotebookTaskDict | CleanRoomsNotebookTask
