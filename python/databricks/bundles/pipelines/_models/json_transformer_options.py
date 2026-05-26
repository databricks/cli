from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.file_ingestion_options_schema_evolution_mode import (
    FileIngestionOptionsSchemaEvolutionMode,
    FileIngestionOptionsSchemaEvolutionModeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JsonTransformerOptions:
    """
    :meta private: [EXPERIMENTAL]
    """

    as_variant: VariableOrOptional[bool] = None
    """
    Parse the entire value as a single Variant column.
    """

    schema: VariableOrOptional[str] = None
    """
    Inline schema string for JSON parsing (Spark DDL format).
    """

    schema_evolution_mode: VariableOrOptional[
        FileIngestionOptionsSchemaEvolutionMode
    ] = None
    """
    (Optional) Schema evolution mode for schema inference.
    """

    schema_file_path: VariableOrOptional[str] = None
    """
    Path to a schema file (.ddl).
    """

    schema_hints: VariableOrOptional[str] = None
    """
    (Optional) Schema hints as a comma-separated string of "column_name type" pairs.
    """

    @classmethod
    def from_dict(cls, value: "JsonTransformerOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JsonTransformerOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class JsonTransformerOptionsDict(TypedDict, total=False):
    """"""

    as_variant: VariableOrOptional[bool]
    """
    Parse the entire value as a single Variant column.
    """

    schema: VariableOrOptional[str]
    """
    Inline schema string for JSON parsing (Spark DDL format).
    """

    schema_evolution_mode: VariableOrOptional[
        FileIngestionOptionsSchemaEvolutionModeParam
    ]
    """
    (Optional) Schema evolution mode for schema inference.
    """

    schema_file_path: VariableOrOptional[str]
    """
    Path to a schema file (.ddl).
    """

    schema_hints: VariableOrOptional[str]
    """
    (Optional) Schema hints as a comma-separated string of "column_name type" pairs.
    """


JsonTransformerOptionsParam = JsonTransformerOptionsDict | JsonTransformerOptions
