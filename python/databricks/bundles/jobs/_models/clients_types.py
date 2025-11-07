from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ClientsTypes:
    """"""

    jobs: VariableOrOptional[bool] = None
    """
    With jobs set, the cluster can be used for jobs
    """

    notebooks: VariableOrOptional[bool] = None
    """
    With notebooks set, this cluster can be used for notebooks
    """

    @classmethod
    def from_dict(cls, value: "ClientsTypesDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ClientsTypesDict":
        return _transform_to_json_value(self)  # type:ignore


class ClientsTypesDict(TypedDict, total=False):
    """"""

    jobs: VariableOrOptional[bool]
    """
    With jobs set, the cluster can be used for jobs
    """

    notebooks: VariableOrOptional[bool]
    """
    With notebooks set, this cluster can be used for notebooks
    """


ClientsTypesParam = ClientsTypesDict | ClientsTypes
