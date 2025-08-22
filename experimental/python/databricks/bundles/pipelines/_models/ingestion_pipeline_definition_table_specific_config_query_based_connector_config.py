from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig:
    """
    :meta private: [EXPERIMENTAL]

    Configurations that are only applicable for query-based ingestion connectors.
    """

    cursor_columns: VariableOrList[str] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    The names of the monotonically increasing columns in the source table that are used to enable
    the table to be read and ingested incrementally through structured streaming.
    The columns are allowed to have repeated values but have to be non-decreasing.
    If the source data is merged into the destination (e.g., using SCD Type 1 or Type 2), these
    columns will implicitly define the `sequence_by` behavior. You can still explicitly set
    `sequence_by` to override this default.
    """

    deletion_condition: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Specifies a SQL WHERE condition that specifies that the source row has been deleted.
    This is sometimes referred to as "soft-deletes".
    For example: "Operation = 'DELETE'" or "is_deleted = true".
    This field is orthogonal to `hard_deletion_sync_interval_in_seconds`,
    one for soft-deletes and the other for hard-deletes.
    See also the hard_deletion_sync_min_interval_in_seconds field for
    handling of "hard deletes" where the source rows are physically removed from the table.
    """

    hard_deletion_sync_min_interval_in_seconds: VariableOrOptional[int] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Specifies the minimum interval (in seconds) between snapshots on primary keys
    for detecting and synchronizing hard deletions—i.e., rows that have been
    physically removed from the source table.
    This interval acts as a lower bound. If ingestion runs less frequently than
    this value, hard deletion synchronization will align with the actual ingestion
    frequency instead of happening more often.
    If not set, hard deletion synchronization via snapshots is disabled.
    This field is mutable and can be updated without triggering a full snapshot.
    """

    @classmethod
    def from_dict(
        cls,
        value: "IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigDict",
    ) -> "Self":
        return _transform(cls, value)

    def as_dict(
        self,
    ) -> "IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigDict(
    TypedDict, total=False
):
    """"""

    cursor_columns: VariableOrList[str]
    """
    :meta private: [EXPERIMENTAL]
    
    The names of the monotonically increasing columns in the source table that are used to enable
    the table to be read and ingested incrementally through structured streaming.
    The columns are allowed to have repeated values but have to be non-decreasing.
    If the source data is merged into the destination (e.g., using SCD Type 1 or Type 2), these
    columns will implicitly define the `sequence_by` behavior. You can still explicitly set
    `sequence_by` to override this default.
    """

    deletion_condition: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    Specifies a SQL WHERE condition that specifies that the source row has been deleted.
    This is sometimes referred to as "soft-deletes".
    For example: "Operation = 'DELETE'" or "is_deleted = true".
    This field is orthogonal to `hard_deletion_sync_interval_in_seconds`,
    one for soft-deletes and the other for hard-deletes.
    See also the hard_deletion_sync_min_interval_in_seconds field for
    handling of "hard deletes" where the source rows are physically removed from the table.
    """

    hard_deletion_sync_min_interval_in_seconds: VariableOrOptional[int]
    """
    :meta private: [EXPERIMENTAL]
    
    Specifies the minimum interval (in seconds) between snapshots on primary keys
    for detecting and synchronizing hard deletions—i.e., rows that have been
    physically removed from the source table.
    This interval acts as a lower bound. If ingestion runs less frequently than
    this value, hard deletion synchronization will align with the actual ingestion
    frequency instead of happening more often.
    If not set, hard deletion synchronization via snapshots is disabled.
    This field is mutable and can be updated without triggering a full snapshot.
    """


IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigParam = (
    IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfigDict
    | IngestionPipelineDefinitionTableSpecificConfigQueryBasedConnectorConfig
)
