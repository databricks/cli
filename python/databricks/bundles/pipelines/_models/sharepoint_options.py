from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.file_ingestion_options import (
    FileIngestionOptions,
    FileIngestionOptionsParam,
)
from databricks.bundles.pipelines._models.sharepoint_options_sharepoint_entity_type import (
    SharepointOptionsSharepointEntityType,
    SharepointOptionsSharepointEntityTypeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SharepointOptions:
    """
    :meta private: [EXPERIMENTAL]
    """

    entity_type: VariableOrOptional[SharepointOptionsSharepointEntityType] = None
    """
    (Optional) The type of SharePoint entity to ingest.
    If not specified, defaults to FILE.
    """

    file_ingestion_options: VariableOrOptional[FileIngestionOptions] = None
    """
    (Optional) File ingestion options for processing files.
    """

    url: VariableOrOptional[str] = None
    """
    Required. The SharePoint URL.
    """

    @classmethod
    def from_dict(cls, value: "SharepointOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SharepointOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class SharepointOptionsDict(TypedDict, total=False):
    """"""

    entity_type: VariableOrOptional[SharepointOptionsSharepointEntityTypeParam]
    """
    (Optional) The type of SharePoint entity to ingest.
    If not specified, defaults to FILE.
    """

    file_ingestion_options: VariableOrOptional[FileIngestionOptionsParam]
    """
    (Optional) File ingestion options for processing files.
    """

    url: VariableOrOptional[str]
    """
    Required. The SharePoint URL.
    """


SharepointOptionsParam = SharepointOptionsDict | SharepointOptions
