from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.compute._models.clients_types import (
    ClientsTypes,
    ClientsTypesParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class WorkloadType:
    """
    Cluster Attributes showing for clusters workload types.
    """

    clients: VariableOr[ClientsTypes]
    """
    defined what type of clients can use the cluster. E.g. Notebooks, Jobs
    """

    @classmethod
    def from_dict(cls, value: "WorkloadTypeDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "WorkloadTypeDict":
        return _transform_to_json_value(self)  # type:ignore


class WorkloadTypeDict(TypedDict, total=False):
    """"""

    clients: VariableOr[ClientsTypesParam]
    """
    defined what type of clients can use the cluster. E.g. Notebooks, Jobs
    """


WorkloadTypeParam = WorkloadTypeDict | WorkloadType
