from enum import Enum
from typing import Literal


class TransformerFormat(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    STRING = "STRING"
    JSON = "JSON"
    AVRO = "AVRO"
    PROTOBUF = "PROTOBUF"


TransformerFormatParam = (
    Literal["STRING", "JSON", "AVRO", "PROTOBUF"] | TransformerFormat
)
