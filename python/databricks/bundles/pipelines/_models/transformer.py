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
    :meta private: [EXPERIMENTAL]

    Specifies how to transform binary data into structured data.
    """

    format: VariableOrOptional[TransformerFormat] = None
    """
    :meta private: [EXPERIMENTAL]
    
    Required: the wire format of the data.
    """

    json_options: VariableOrOptional[JsonTransformerOptions] = None
    """
    :meta private: [EXPERIMENTAL]
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
    :meta private: [EXPERIMENTAL]
    
    Required: the wire format of the data.
    """

    json_options: VariableOrOptional[JsonTransformerOptionsParam]
    """
    :meta private: [EXPERIMENTAL]
    """


TransformerParam = TransformerDict | Transformer
