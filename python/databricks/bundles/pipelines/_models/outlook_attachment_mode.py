from enum import Enum
from typing import Literal


class OutlookAttachmentMode(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Attachment behavior mode for Outlook ingestion
    """

    ALL = "ALL"
    NON_INLINE_ONLY = "NON_INLINE_ONLY"
    INLINE_ONLY = "INLINE_ONLY"
    NONE = "NONE"


OutlookAttachmentModeParam = (
    Literal["ALL", "NON_INLINE_ONLY", "INLINE_ONLY", "NONE"] | OutlookAttachmentMode
)
