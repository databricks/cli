from enum import Enum
from typing import Literal


class OutlookBodyFormat(Enum):
    """
    :meta private: [EXPERIMENTAL]

    Body format for Outlook email content
    """

    TEXT_HTML = "TEXT_HTML"
    TEXT_PLAIN = "TEXT_PLAIN"


OutlookBodyFormatParam = Literal["TEXT_HTML", "TEXT_PLAIN"] | OutlookBodyFormat
