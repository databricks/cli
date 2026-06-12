from enum import Enum
from typing import Literal


class TransformerFormat(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    STRING = "STRING"
    JSON = "JSON"


TransformerFormatParam = Literal["STRING", "JSON"] | TransformerFormat
