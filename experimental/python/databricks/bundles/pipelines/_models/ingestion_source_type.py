from enum import Enum
from typing import Literal


class IngestionSourceType(Enum):
    MYSQL = "MYSQL"
    POSTGRESQL = "POSTGRESQL"
    SQLSERVER = "SQLSERVER"
    SALESFORCE = "SALESFORCE"
    BIGQUERY = "BIGQUERY"
    NETSUITE = "NETSUITE"
    WORKDAY_RAAS = "WORKDAY_RAAS"
    GA4_RAW_DATA = "GA4_RAW_DATA"
    SERVICENOW = "SERVICENOW"
    MANAGED_POSTGRESQL = "MANAGED_POSTGRESQL"
    ORACLE = "ORACLE"
    TERADATA = "TERADATA"
    SHAREPOINT = "SHAREPOINT"
    DYNAMICS365 = "DYNAMICS365"
    CONFLUENCE = "CONFLUENCE"


IngestionSourceTypeParam = (
    Literal[
        "MYSQL",
        "POSTGRESQL",
        "SQLSERVER",
        "SALESFORCE",
        "BIGQUERY",
        "NETSUITE",
        "WORKDAY_RAAS",
        "GA4_RAW_DATA",
        "SERVICENOW",
        "MANAGED_POSTGRESQL",
        "ORACLE",
        "TERADATA",
        "SHAREPOINT",
        "DYNAMICS365",
        "CONFLUENCE",
    ]
    | IngestionSourceType
)
