from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class RunAs:
    """
    :meta private: [EXPERIMENTAL]

    Write-only setting, available only in Create/Update calls. Specifies the user or service principal that the pipeline runs as. If not specified, the pipeline runs as the user who created the pipeline.

    Only `user_name` or `service_principal_name` can be specified. If both are specified, an error is thrown.
    """

    service_principal_name: VariableOrOptional[str] = None
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str] = None
    """
    The email of an active workspace user. Users can only set this field to their own email.
    """

    @classmethod
    def from_dict(cls, value: "RunAsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "RunAsDict":
        return _transform_to_json_value(self)  # type:ignore


class RunAsDict(TypedDict, total=False):
    """"""

    service_principal_name: VariableOrOptional[str]
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str]
    """
    The email of an active workspace user. Users can only set this field to their own email.
    """


RunAsParam = RunAsDict | RunAs
