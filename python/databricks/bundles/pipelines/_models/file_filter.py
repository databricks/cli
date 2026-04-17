from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class FileFilter:
    """
    :meta private: [EXPERIMENTAL]
    """

    modified_after: VariableOrOptional[str] = None
    """
    Include files with modification times occurring after the specified time.
    Timestamp format: YYYY-MM-DDTHH:mm:ss (e.g. 2020-06-01T13:00:00)
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#modification-time-path-filters
    """

    modified_before: VariableOrOptional[str] = None
    """
    Include files with modification times occurring before the specified time.
    Timestamp format: YYYY-MM-DDTHH:mm:ss (e.g. 2020-06-01T13:00:00)
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#modification-time-path-filters
    """

    path_filter: VariableOrOptional[str] = None
    """
    Include files with file names matching the pattern
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#path-glob-filter
    """

    @classmethod
    def from_dict(cls, value: "FileFilterDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FileFilterDict":
        return _transform_to_json_value(self)  # type:ignore


class FileFilterDict(TypedDict, total=False):
    """"""

    modified_after: VariableOrOptional[str]
    """
    Include files with modification times occurring after the specified time.
    Timestamp format: YYYY-MM-DDTHH:mm:ss (e.g. 2020-06-01T13:00:00)
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#modification-time-path-filters
    """

    modified_before: VariableOrOptional[str]
    """
    Include files with modification times occurring before the specified time.
    Timestamp format: YYYY-MM-DDTHH:mm:ss (e.g. 2020-06-01T13:00:00)
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#modification-time-path-filters
    """

    path_filter: VariableOrOptional[str]
    """
    Include files with file names matching the pattern
    Based on https://spark.apache.org/docs/latest/sql-data-sources-generic-options.html#path-glob-filter
    """


FileFilterParam = FileFilterDict | FileFilter
