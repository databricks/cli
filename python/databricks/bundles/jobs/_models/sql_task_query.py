from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SqlTaskQuery:
    """"""

    query_id: VariableOr[str]
    """
    The canonical identifier of the SQL query.
    """

    @classmethod
    def from_dict(cls, value: "SqlTaskQueryDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SqlTaskQueryDict":
        return _transform_to_json_value(self)  # type:ignore


class SqlTaskQueryDict(TypedDict, total=False):
    """"""

    query_id: VariableOr[str]
    """
    The canonical identifier of the SQL query.
    """


SqlTaskQueryParam = SqlTaskQueryDict | SqlTaskQuery
