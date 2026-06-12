from enum import Enum
from typing import Literal


class FileIngestionOptionsFileFormat(Enum):
    """
    :meta private: [EXPERIMENTAL]
    """

    BINARYFILE = "BINARYFILE"
    JSON = "JSON"
    CSV = "CSV"
    XML = "XML"
    EXCEL = "EXCEL"
    PARQUET = "PARQUET"
    AVRO = "AVRO"
    ORC = "ORC"


FileIngestionOptionsFileFormatParam = (
    Literal["BINARYFILE", "JSON", "CSV", "XML", "EXCEL", "PARQUET", "AVRO", "ORC"]
    | FileIngestionOptionsFileFormat
)
