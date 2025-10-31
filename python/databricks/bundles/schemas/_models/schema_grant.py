from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList
from databricks.bundles.schemas._models.schema_grant_privilege import (
    SchemaGrantPrivilege,
    SchemaGrantPrivilegeParam,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class SchemaGrant:
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[SchemaGrantPrivilege] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "SchemaGrantDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "SchemaGrantDict":
        return _transform_to_json_value(self)  # type:ignore


class SchemaGrantDict(TypedDict, total=False):
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[SchemaGrantPrivilegeParam]


SchemaGrantParam = SchemaGrantDict | SchemaGrant
