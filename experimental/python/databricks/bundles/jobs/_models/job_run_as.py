from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class JobRunAs:
    """
    Write-only setting. Specifies the user or service principal that the job runs as. If not specified, the job runs as the user who created the job.

    Either `user_name` or `service_principal_name` should be specified. If not, an error is thrown.
    """

    service_principal_name: VariableOrOptional[str] = None
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str] = None
    """
    The email of an active workspace user. Non-admin users can only set this field to their own email.
    """

    def __post_init__(self):
        union_fields = [
            self.user_name,
            self.service_principal_name,
        ]

        if sum(f is not None for f in union_fields) != 1:
            raise ValueError(
                "JobRunAs must specify exactly one of 'user_name', 'service_principal_name'"
            )

    @classmethod
    def from_dict(cls, value: "JobRunAsDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "JobRunAsDict":
        return _transform_to_json_value(self)  # type:ignore


class JobRunAsDict(TypedDict, total=False):
    """"""

    service_principal_name: VariableOrOptional[str]
    """
    Application ID of an active service principal. Setting this field requires the `servicePrincipal/user` role.
    """

    user_name: VariableOrOptional[str]
    """
    The email of an active workspace user. Non-admin users can only set this field to their own email.
    """


JobRunAsParam = JobRunAsDict | JobRunAs
