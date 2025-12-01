from enum import Enum
from typing import Literal


class VolumeType(Enum):
    """
    The type of the volume. An external volume is located in the specified external location. A managed volume is located in the default location which is specified by the parent schema, or the parent catalog, or the Metastore. [Learn more](https://docs.databricks.com/aws/en/volumes/managed-vs-external)
    """

    EXTERNAL = "EXTERNAL"
    MANAGED = "MANAGED"


VolumeTypeParam = Literal["EXTERNAL", "MANAGED"] | VolumeType
