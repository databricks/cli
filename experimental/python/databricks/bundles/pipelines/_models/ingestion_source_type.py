from enum import Enum
from typing import Literal


class IngestionSourceType(Enum):
    MYSQL = "MYSQL"
    POSTGRESQL = "POSTGRESQL"
    SQLSERVER = "SQLSERVER"
    SALESFORCE = "SALESFORCE"
    NETSUITE = "NETSUITE"
    WORKDAY_RAAS = "WORKDAY_RAAS"
    GA4_RAW_DATA = "GA4_RAW_DATA"
    SERVICENOW = "SERVICENOW"
    MANAGED_POSTGRESQL = "MANAGED_POSTGRESQL"
    ORACLE = "ORACLE"
    SHAREPOINT = "SHAREPOINT"
    DYNAMICS365 = "DYNAMICS365"


IngestionSourceTypeParam = (
    Literal[
        "MYSQL",
        "POSTGRESQL",
        "SQLSERVER",
        "SALESFORCE",
        "NETSUITE",
        "WORKDAY_RAAS",
        "GA4_RAW_DATA",
        "SERVICENOW",
        "MANAGED_POSTGRESQL",
        "ORACLE",
        "SHAREPOINT",
        "DYNAMICS365",
    ]
    | IngestionSourceType
)
