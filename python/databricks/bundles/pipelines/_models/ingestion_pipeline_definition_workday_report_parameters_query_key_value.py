from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue:
    """
    :meta private: [EXPERIMENTAL]

    [DEPRECATED]
    """

    key: VariableOrOptional[str] = None
    """
    Key for the report parameter, can be a column name or other metadata
    """

    value: VariableOrOptional[str] = None
    """
    Value for the report parameter.
    Possible values it can take are these sql functions:
    1. coalesce(current_offset(), date("YYYY-MM-DD")) -> if current_offset() is null, then the passed date, else current_offset()
    2. current_date()
    3. date_sub(current_date(), x) -> subtract x (some non-negative integer) days from current date
    """

    @classmethod
    def from_dict(
        cls,
        value: "IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueDict",
    ) -> "Self":
        return _transform(cls, value)

    def as_dict(
        self,
    ) -> "IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueDict(
    TypedDict, total=False
):
    """"""

    key: VariableOrOptional[str]
    """
    Key for the report parameter, can be a column name or other metadata
    """

    value: VariableOrOptional[str]
    """
    Value for the report parameter.
    Possible values it can take are these sql functions:
    1. coalesce(current_offset(), date("YYYY-MM-DD")) -> if current_offset() is null, then the passed date, else current_offset()
    2. current_date()
    3. date_sub(current_date(), x) -> subtract x (some non-negative integer) days from current date
    """


IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueParam = (
    IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValueDict
    | IngestionPipelineDefinitionWorkdayReportParametersQueryKeyValue
)
