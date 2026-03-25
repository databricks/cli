from enum import Enum
from typing import Literal


class ConnectorType(Enum):
    """
    :meta private: [EXPERIMENTAL]

    For certain database sources LakeFlow Connect offers both query based and cdc
    ingestion, ConnectorType can bse used to convey the type of ingestion.
    If connection_name is provided for database sources, we default to Query Based ingestion
    """

    CDC = "CDC"
    QUERY_BASED = "QUERY_BASED"


ConnectorTypeParam = Literal["CDC", "QUERY_BASED"] | ConnectorType
