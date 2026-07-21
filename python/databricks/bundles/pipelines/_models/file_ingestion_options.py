from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.file_filter import FileFilter, FileFilterParam
from databricks.bundles.pipelines._models.file_ingestion_options_file_format import (
    FileIngestionOptionsFileFormat,
    FileIngestionOptionsFileFormatParam,
)
from databricks.bundles.pipelines._models.file_ingestion_options_schema_evolution_mode import (
    FileIngestionOptionsSchemaEvolutionMode,
    FileIngestionOptionsSchemaEvolutionModeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class FileIngestionOptions:
    """
    :meta private: [EXPERIMENTAL]
    """

    corrupt_record_column: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    file_filters: VariableOrList[FileFilter] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Generic options
    """

    format: VariableOrOptional[FileIngestionOptionsFileFormat] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] required for TableSpec
    """

    format_options: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Format-specific options
    Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/options#file-format-options
    """

    ignore_corrupt_files: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    infer_column_types: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    reader_case_sensitive: VariableOrOptional[bool] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Column name case sensitivity
    https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#change-case-sensitive-behavior
    """

    rescued_data_column: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    schema_evolution_mode: VariableOrOptional[
        FileIngestionOptionsSchemaEvolutionMode
    ] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#how-does-auto-loader-schema-evolution-work
    """

    schema_hints: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Override inferred schema of specific columns
    Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#override-schema-inference-with-schema-hints
    """

    single_variant_column: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    @classmethod
    def from_dict(cls, value: "FileIngestionOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "FileIngestionOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class FileIngestionOptionsDict(TypedDict, total=False):
    """"""

    corrupt_record_column: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    file_filters: VariableOrList[FileFilterParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Generic options
    """

    format: VariableOrOptional[FileIngestionOptionsFileFormatParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] required for TableSpec
    """

    format_options: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Format-specific options
    Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/options#file-format-options
    """

    ignore_corrupt_files: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    infer_column_types: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    reader_case_sensitive: VariableOrOptional[bool]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Column name case sensitivity
    https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#change-case-sensitive-behavior
    """

    rescued_data_column: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """

    schema_evolution_mode: VariableOrOptional[
        FileIngestionOptionsSchemaEvolutionModeParam
    ]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#how-does-auto-loader-schema-evolution-work
    """

    schema_hints: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Override inferred schema of specific columns
    Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#override-schema-inference-with-schema-hints
    """

    single_variant_column: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview]
    """


FileIngestionOptionsParam = FileIngestionOptionsDict | FileIngestionOptions
