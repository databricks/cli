from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.file_ingestion_options import (
    FileIngestionOptions,
    FileIngestionOptionsParam,
)
from databricks.bundles.pipelines._models.google_drive_options_google_drive_entity_type import (
    GoogleDriveOptionsGoogleDriveEntityType,
    GoogleDriveOptionsGoogleDriveEntityTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class GoogleDriveOptions:
    """
    :meta private: [EXPERIMENTAL]
    """

    entity_type: VariableOrOptional[GoogleDriveOptionsGoogleDriveEntityType] = None

    file_ingestion_options: VariableOrOptional[FileIngestionOptions] = None

    url: VariableOrOptional[str] = None
    """
    Google Drive URL.
    """

    @classmethod
    def from_dict(cls, value: "GoogleDriveOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "GoogleDriveOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class GoogleDriveOptionsDict(TypedDict, total=False):
    """"""

    entity_type: VariableOrOptional[GoogleDriveOptionsGoogleDriveEntityTypeParam]

    file_ingestion_options: VariableOrOptional[FileIngestionOptionsParam]

    url: VariableOrOptional[str]
    """
    Google Drive URL.
    """


GoogleDriveOptionsParam = GoogleDriveOptionsDict | GoogleDriveOptions
