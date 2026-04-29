from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrList, VariableOrOptional
from databricks.bundles.pipelines._models.outlook_attachment_mode import (
    OutlookAttachmentMode,
    OutlookAttachmentModeParam,
)
from databricks.bundles.pipelines._models.outlook_body_format import (
    OutlookBodyFormat,
    OutlookBodyFormatParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class OutlookOptions:
    """
    :meta private: [EXPERIMENTAL]

    Outlook specific options for ingestion
    """

    attachment_mode: VariableOrOptional[OutlookAttachmentMode] = None
    """
    (Optional) Controls which attachments to ingest.
    If not specified, defaults to ALL.
    """

    body_format: VariableOrOptional[OutlookBodyFormat] = None
    """
    (Optional) Defines how the body_content column is populated.
    TEXT_HTML: Preserves full formatting, links, and styling.
    TEXT_PLAIN: Converts body to plain text. Recommended for AI/RAG pipelines to reduce token usage and noise.
    """

    folder_filter: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] Deprecated. Use include_folders instead.
    """

    include_folders: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Filter mail folders to include in the sync.
    If not specified, all folders will be synced.
    Examples: Inbox, Sent Items, Custom_Folder
    Filter semantics: OR between different folders.
    """

    include_mailboxes: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) List of mailboxes to sync (e.g. mailbox email addresses or identifiers).
    If not specified, all accessible mailboxes are ingested.
    Filter semantics: OR between different mailboxes.
    """

    include_senders: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Filter emails by sender address. Uses exact email match.
    Examples: user@vendor.com, alerts@system.io, noreply@company.com
    If not specified, emails from all senders will be synced.
    Filter semantics: OR between different senders.
    """

    include_subjects: VariableOrList[str] = field(default_factory=list)
    """
    (Optional) Filter emails by subject line. Values ending with "*" use prefix match (subject starts with
    the part before "*"); otherwise substring match (subject contains the value).
    Examples: "Invoice" (substring), "Re:*" (prefix), "Support Ticket", "URGENT*"
    If not specified, emails with all subjects will be synced.
    Filter semantics: OR between different subjects.
    """

    sender_filter: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] Deprecated. Use include_senders instead.
    """

    start_date: VariableOrOptional[str] = None
    """
    (Optional) Start date for the initial sync in YYYY-MM-DD format.
    Format: YYYY-MM-DD (e.g., 2024-01-01)
    This determines the earliest date from which to sync historical data.
    If not specified, complete history is ingested.
    """

    subject_filter: VariableOrList[str] = field(default_factory=list)
    """
    [DEPRECATED] Deprecated. Use include_subjects instead.
    """

    @classmethod
    def from_dict(cls, value: "OutlookOptionsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "OutlookOptionsDict":
        return _transform_to_json_value(self)  # type:ignore


class OutlookOptionsDict(TypedDict, total=False):
    """"""

    attachment_mode: VariableOrOptional[OutlookAttachmentModeParam]
    """
    (Optional) Controls which attachments to ingest.
    If not specified, defaults to ALL.
    """

    body_format: VariableOrOptional[OutlookBodyFormatParam]
    """
    (Optional) Defines how the body_content column is populated.
    TEXT_HTML: Preserves full formatting, links, and styling.
    TEXT_PLAIN: Converts body to plain text. Recommended for AI/RAG pipelines to reduce token usage and noise.
    """

    folder_filter: VariableOrList[str]
    """
    [DEPRECATED] Deprecated. Use include_folders instead.
    """

    include_folders: VariableOrList[str]
    """
    (Optional) Filter mail folders to include in the sync.
    If not specified, all folders will be synced.
    Examples: Inbox, Sent Items, Custom_Folder
    Filter semantics: OR between different folders.
    """

    include_mailboxes: VariableOrList[str]
    """
    (Optional) List of mailboxes to sync (e.g. mailbox email addresses or identifiers).
    If not specified, all accessible mailboxes are ingested.
    Filter semantics: OR between different mailboxes.
    """

    include_senders: VariableOrList[str]
    """
    (Optional) Filter emails by sender address. Uses exact email match.
    Examples: user@vendor.com, alerts@system.io, noreply@company.com
    If not specified, emails from all senders will be synced.
    Filter semantics: OR between different senders.
    """

    include_subjects: VariableOrList[str]
    """
    (Optional) Filter emails by subject line. Values ending with "*" use prefix match (subject starts with
    the part before "*"); otherwise substring match (subject contains the value).
    Examples: "Invoice" (substring), "Re:*" (prefix), "Support Ticket", "URGENT*"
    If not specified, emails with all subjects will be synced.
    Filter semantics: OR between different subjects.
    """

    sender_filter: VariableOrList[str]
    """
    [DEPRECATED] Deprecated. Use include_senders instead.
    """

    start_date: VariableOrOptional[str]
    """
    (Optional) Start date for the initial sync in YYYY-MM-DD format.
    Format: YYYY-MM-DD (e.g., 2024-01-01)
    This determines the earliest date from which to sync historical data.
    If not specified, complete history is ingested.
    """

    subject_filter: VariableOrList[str]
    """
    [DEPRECATED] Deprecated. Use include_subjects instead.
    """


OutlookOptionsParam = OutlookOptionsDict | OutlookOptions
