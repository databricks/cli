from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_uc_securable_uc_securable_permission import (
    AppResourceUcSecurableUcSecurablePermission,
    AppResourceUcSecurableUcSecurablePermissionParam,
)
from databricks.bundles.apps._models.app_resource_uc_securable_uc_securable_type import (
    AppResourceUcSecurableUcSecurableType,
    AppResourceUcSecurableUcSecurableTypeParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceUcSecurable:
    """"""

    permission: VariableOr[AppResourceUcSecurableUcSecurablePermission]

    securable_full_name: VariableOr[str]

    securable_type: VariableOr[AppResourceUcSecurableUcSecurableType]

    @classmethod
    def from_dict(cls, value: "AppResourceUcSecurableDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceUcSecurableDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceUcSecurableDict(TypedDict, total=False):
    """"""

    permission: VariableOr[AppResourceUcSecurableUcSecurablePermissionParam]

    securable_full_name: VariableOr[str]

    securable_type: VariableOr[AppResourceUcSecurableUcSecurableTypeParam]


AppResourceUcSecurableParam = AppResourceUcSecurableDict | AppResourceUcSecurable
