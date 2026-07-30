from enum import Enum
from typing import Literal


class JobsHealthMetric(Enum):
    """
    Specifies the health metric that is being evaluated for a particular health rule.

    * `RUN_DURATION_SECONDS`: Expected total time for a run in seconds.
    * `STREAMING_BACKLOG_BYTES`: An estimate of the maximum bytes of data waiting to be consumed across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_RECORDS`: An estimate of the maximum offset lag across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_SECONDS`: An estimate of the maximum consumer delay across all streams. This metric is in Public Preview.
    * `STREAMING_BACKLOG_FILES`: An estimate of the maximum number of outstanding files across all streams. This metric is in Public Preview.
    """

    RUN_DURATION_SECONDS = "RUN_DURATION_SECONDS"
    STREAMING_BACKLOG_BYTES = "STREAMING_BACKLOG_BYTES"
    STREAMING_BACKLOG_RECORDS = "STREAMING_BACKLOG_RECORDS"
    STREAMING_BACKLOG_SECONDS = "STREAMING_BACKLOG_SECONDS"
    STREAMING_BACKLOG_FILES = "STREAMING_BACKLOG_FILES"


JobsHealthMetricParam = (
    Literal[
        "RUN_DURATION_SECONDS",
        "STREAMING_BACKLOG_BYTES",
        "STREAMING_BACKLOG_RECORDS",
        "STREAMING_BACKLOG_SECONDS",
        "STREAMING_BACKLOG_FILES",
    ]
    | JobsHealthMetric
)
