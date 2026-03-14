from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AutoFullRefreshPolicy:
    """
    Policy for auto full refresh.
    """

    enabled: VariableOr[bool]
    """
    (Required, Mutable) Whether to enable auto full refresh or not.
    """

    min_interval_hours: VariableOrOptional[int] = None
    """
    (Optional, Mutable) Specify the minimum interval in hours between the timestamp
    at which a table was last full refreshed and the current timestamp for triggering auto full
    If unspecified and autoFullRefresh is enabled then by default min_interval_hours is 24 hours.
    """

    @classmethod
    def from_dict(cls, value: "AutoFullRefreshPolicyDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AutoFullRefreshPolicyDict":
        return _transform_to_json_value(self)  # type:ignore


class AutoFullRefreshPolicyDict(TypedDict, total=False):
    """"""

    enabled: VariableOr[bool]
    """
    (Required, Mutable) Whether to enable auto full refresh or not.
    """

    min_interval_hours: VariableOrOptional[int]
    """
    (Optional, Mutable) Specify the minimum interval in hours between the timestamp
    at which a table was last full refreshed and the current timestamp for triggering auto full
    If unspecified and autoFullRefresh is enabled then by default min_interval_hours is 24 hours.
    """


AutoFullRefreshPolicyParam = AutoFullRefreshPolicyDict | AutoFullRefreshPolicy
