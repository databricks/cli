from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CustomTag:
    """"""

    key: VariableOrOptional[str] = None
    """
    The key of the custom tag.
    """

    value: VariableOrOptional[str] = None
    """
    The value of the custom tag.
    """

    @classmethod
    def from_dict(cls, value: "CustomTagDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CustomTagDict":
        return _transform_to_json_value(self)  # type:ignore


class CustomTagDict(TypedDict, total=False):
    """"""

    key: VariableOrOptional[str]
    """
    The key of the custom tag.
    """

    value: VariableOrOptional[str]
    """
    The value of the custom tag.
    """


CustomTagParam = CustomTagDict | CustomTag
