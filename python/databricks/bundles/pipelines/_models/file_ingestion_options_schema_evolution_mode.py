from enum import Enum
from typing import Literal


class FileIngestionOptionsSchemaEvolutionMode(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Based on https://docs.databricks.com/aws/en/ingestion/cloud-object-storage/auto-loader/schema#how-does-auto-loader-schema-evolution-work
    """

    ADD_NEW_COLUMNS_WITH_TYPE_WIDENING = "ADD_NEW_COLUMNS_WITH_TYPE_WIDENING"
    ADD_NEW_COLUMNS = "ADD_NEW_COLUMNS"
    RESCUE = "RESCUE"
    FAIL_ON_NEW_COLUMNS = "FAIL_ON_NEW_COLUMNS"
    NONE = "NONE"


FileIngestionOptionsSchemaEvolutionModeParam = (
    Literal[
        "ADD_NEW_COLUMNS_WITH_TYPE_WIDENING",
        "ADD_NEW_COLUMNS",
        "RESCUE",
        "FAIL_ON_NEW_COLUMNS",
        "NONE",
    ]
    | FileIngestionOptionsSchemaEvolutionMode
)
