from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class DatabaseInstanceRef:
    """
    DatabaseInstanceRef is a reference to a database instance. It is used in the
    DatabaseInstance object to refer to the parent instance of an instance and
    to refer the child instances of an instance.
    To specify as a parent instance during creation of an instance,
    the lsn and branch_time fields are optional. If not specified, the child
    instance will be created from the latest lsn of the parent.
    If both lsn and branch_time are specified, the lsn will be used to create
    the child instance.
    """

    branch_time: VariableOrOptional[str] = None
    """
    Branch time of the ref database instance.
    For a parent ref instance, this is the point in time on the parent instance from which the
    instance was created.
    For a child ref instance, this is the point in time on the instance from which the child
    instance was created.
    Input: For specifying the point in time to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    effective_lsn: VariableOrOptional[str] = None
    """
    For a parent ref instance, this is the LSN on the parent instance from which the
    instance was created.
    For a child ref instance, this is the LSN on the instance from which the child instance
    was created.
    """

    lsn: VariableOrOptional[str] = None
    """
    User-specified WAL LSN of the ref database instance.
    
    Input: For specifying the WAL LSN to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    name: VariableOrOptional[str] = None
    """
    Name of the ref database instance.
    """

    uid: VariableOrOptional[str] = None
    """
    Id of the ref database instance.
    """

    @classmethod
    def from_dict(cls, value: "DatabaseInstanceRefDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "DatabaseInstanceRefDict":
        return _transform_to_json_value(self)  # type:ignore


class DatabaseInstanceRefDict(TypedDict, total=False):
    """"""

    branch_time: VariableOrOptional[str]
    """
    Branch time of the ref database instance.
    For a parent ref instance, this is the point in time on the parent instance from which the
    instance was created.
    For a child ref instance, this is the point in time on the instance from which the child
    instance was created.
    Input: For specifying the point in time to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    effective_lsn: VariableOrOptional[str]
    """
    For a parent ref instance, this is the LSN on the parent instance from which the
    instance was created.
    For a child ref instance, this is the LSN on the instance from which the child instance
    was created.
    """

    lsn: VariableOrOptional[str]
    """
    User-specified WAL LSN of the ref database instance.
    
    Input: For specifying the WAL LSN to create a child instance. Optional.
    Output: Only populated if provided as input to create a child instance.
    """

    name: VariableOrOptional[str]
    """
    Name of the ref database instance.
    """

    uid: VariableOrOptional[str]
    """
    Id of the ref database instance.
    """


DatabaseInstanceRefParam = DatabaseInstanceRefDict | DatabaseInstanceRef
