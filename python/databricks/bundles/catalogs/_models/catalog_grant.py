from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.catalogs._models.catalog_grant_privilege import (
    CatalogGrantPrivilege,
    CatalogGrantPrivilegeParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrList

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class CatalogGrant:
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[CatalogGrantPrivilege] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "CatalogGrantDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CatalogGrantDict":
        return _transform_to_json_value(self)  # type:ignore


class CatalogGrantDict(TypedDict, total=False):
    """"""

    principal: VariableOr[str]

    privileges: VariableOrList[CatalogGrantPrivilegeParam]


CatalogGrantParam = CatalogGrantDict | CatalogGrant
