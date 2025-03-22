from enum import Enum
from typing import Literal


class RuntimeEngine(Enum):
    """
    Determines the cluster's runtime engine, either standard or Photon.

    This field is not compatible with legacy `spark_version` values that contain `-photon-`.
    Remove `-photon-` from the `spark_version` and set `runtime_engine` to `PHOTON`.

    If left unspecified, the runtime engine defaults to standard unless the spark_version
    contains -photon-, in which case Photon will be used.

    """

    NULL = "NULL"
    STANDARD = "STANDARD"
    PHOTON = "PHOTON"


RuntimeEngineParam = Literal["NULL", "STANDARD", "PHOTON"] | RuntimeEngine
