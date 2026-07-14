from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.pipelines._models.transformer import (
    Transformer,
    TransformerParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class KafkaOptions:
    """
    :meta private: [EXPERIMENTAL]
    """

    client_config: VariableOrDict[str] = field(default_factory=dict)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Undocumented backdoor mechanism for overriding parameters
    to pass to the Kafka client.
    This is not supported and may break at any time.
    """

    key_transformer: VariableOrOptional[Transformer] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Transformer for the message key.
    If not specified, the key is left as raw bytes.
    """

    max_offsets_per_trigger: VariableOrOptional[int] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Internal option to control the maximum number of offsets to process per trigger.
    """

    starting_offset: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Where to begin reading when no checkpoint exists.
    Valid values: "latest" and "earliest". Defaults to "latest".
    """

    topic_pattern: VariableOrOptional[str] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Java regex pattern to subscribe to matching topics.
    Only one of topics or topic_pattern must be specified.
    """

    topics: VariableOrList[str] = field(default_factory=list)
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Topics to subscribe to.
    Only one of topics or topic_pattern must be specified.
    """

    value_transformer: VariableOrOptional[Transformer] = None
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Transformer for the message value.
    If not specified, the value is left as raw bytes.
    """

    @classmethod
    def from_dict(cls, value: "KafkaOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "KafkaOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class KafkaOptionsDict(TypedDict, total=False):
    """"""

    client_config: VariableOrDict[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Undocumented backdoor mechanism for overriding parameters
    to pass to the Kafka client.
    This is not supported and may break at any time.
    """

    key_transformer: VariableOrOptional[TransformerParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Transformer for the message key.
    If not specified, the key is left as raw bytes.
    """

    max_offsets_per_trigger: VariableOrOptional[int]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Internal option to control the maximum number of offsets to process per trigger.
    """

    starting_offset: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Where to begin reading when no checkpoint exists.
    Valid values: "latest" and "earliest". Defaults to "latest".
    """

    topic_pattern: VariableOrOptional[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Java regex pattern to subscribe to matching topics.
    Only one of topics or topic_pattern must be specified.
    """

    topics: VariableOrList[str]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] Topics to subscribe to.
    Only one of topics or topic_pattern must be specified.
    """

    value_transformer: VariableOrOptional[TransformerParam]
    """
    :meta private: [EXPERIMENTAL]
    
    [Private Preview] (Optional) Transformer for the message value.
    If not specified, the value is left as raw bytes.
    """


KafkaOptionsParam = KafkaOptionsDict | KafkaOptions
