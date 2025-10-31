from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.app_resource_secret_secret_permission import (
    AppResourceSecretSecretPermission,
    AppResourceSecretSecretPermissionParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AppResourceSecret:
    """"""

    key: VariableOr[str]

    permission: VariableOr[AppResourceSecretSecretPermission]

    scope: VariableOr[str]

    @classmethod
    def from_dict(cls, value: "AppResourceSecretDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AppResourceSecretDict":
        return _transform_to_json_value(self)  # type:ignore


class AppResourceSecretDict(TypedDict, total=False):
    """"""

    key: VariableOr[str]

    permission: VariableOr[AppResourceSecretSecretPermissionParam]

    scope: VariableOr[str]


AppResourceSecretParam = AppResourceSecretDict | AppResourceSecret
