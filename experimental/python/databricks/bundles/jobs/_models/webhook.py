from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Webhook:
    """"""

    id: VariableOr[str]

    @classmethod
    def from_dict(cls, value: "WebhookDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "WebhookDict":
        return _transform_to_json_value(self)  # type:ignore


class WebhookDict(TypedDict, total=False):
    """"""

    id: VariableOr[str]


WebhookParam = WebhookDict | Webhook
