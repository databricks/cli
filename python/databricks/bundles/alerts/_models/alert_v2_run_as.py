from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class AlertV2RunAs:
    """"""

    service_principal_name: VariableOrOptional[str] = None
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str] = None
    """
    The email of an active workspace user. Can only set this field to their own email.
    """

    @classmethod
    def from_dict(cls, value: "AlertV2RunAsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "AlertV2RunAsDict":
        return _transform_to_json_value(self)  # type:ignore


class AlertV2RunAsDict(TypedDict, total=False):
    """"""

    service_principal_name: VariableOrOptional[str]
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str]
    """
    The email of an active workspace user. Can only set this field to their own email.
    """


AlertV2RunAsParam = AlertV2RunAsDict | AlertV2RunAs
