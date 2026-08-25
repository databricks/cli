from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.transformer import (
    Transformer,
    TransformerParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class IngestionPipelineDefinitionFanoutOptions:
    """
    Fanout configuration for multi-table routing from streaming sources.
    Routes each input record to a destination table based on a routing
    key derived from the record. The key value becomes the table name
    suffix: {destination_catalog}.{destination_schema}.{key_value}.
    """

    fanout_by: VariableOrOptional[str] = None
    """
    [Beta] Column path or SQL expression whose value determines the destination table.
    Supports dotted paths (e.g. "value.event_name") and expressions
    (e.g. "value:event_name::string").
    """

    transforms: VariableOrList[Transformer] = field(default_factory=list)
    """
    [Beta] Optional transforms applied to each route's DataFrame before writing
    to the destination table.
    """

    @classmethod
    def from_dict(cls, value: "IngestionPipelineDefinitionFanoutOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "IngestionPipelineDefinitionFanoutOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class IngestionPipelineDefinitionFanoutOptionsDict(TypedDict, total=False):
    """"""

    fanout_by: VariableOrOptional[str]
    """
    [Beta] Column path or SQL expression whose value determines the destination table.
    Supports dotted paths (e.g. "value.event_name") and expressions
    (e.g. "value:event_name::string").
    """

    transforms: VariableOrList[TransformerParam]
    """
    [Beta] Optional transforms applied to each route's DataFrame before writing
    to the destination table.
    """


IngestionPipelineDefinitionFanoutOptionsParam = (
    IngestionPipelineDefinitionFanoutOptionsDict
    | IngestionPipelineDefinitionFanoutOptions
)
