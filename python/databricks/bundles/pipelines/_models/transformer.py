from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional
from databricks.bundles.pipelines._models.json_transformer_options import (
    JsonTransformerOptions,
    JsonTransformerOptionsParam,
)
from databricks.bundles.pipelines._models.transformer_format import (
    TransformerFormat,
    TransformerFormatParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Transformer:
    """
    Specifies how to transform binary data into structured data.
    """

    format: VariableOrOptional[TransformerFormat] = None
    """
    [Beta] Required: the wire format of the data.
    """

    input_column: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional input column to transform. When set, the transformer reads
    from this column instead of the default source column.
    """

    json_options: VariableOrOptional[JsonTransformerOptions] = None
    """
    [Beta]
    """

    output_column: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional output column name. When set, the transformed result is
    written to this column instead of replacing the input column.
    """

    @classmethod
    def from_dict(cls, value: "TransformerDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "TransformerDict":
        return _transform_to_json_value(self)  # type:ignore


class TransformerDict(TypedDict, total=False):
    """"""

    format: VariableOrOptional[TransformerFormatParam]
    """
    [Beta] Required: the wire format of the data.
    """

    input_column: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional input column to transform. When set, the transformer reads
    from this column instead of the default source column.
    """

    json_options: VariableOrOptional[JsonTransformerOptionsParam]
    """
    [Beta]
    """

    output_column: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Optional output column name. When set, the transformed result is
    written to this column instead of replacing the input column.
    """


TransformerParam = TransformerDict | Transformer
