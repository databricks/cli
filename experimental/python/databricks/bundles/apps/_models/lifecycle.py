from dataclasses import dataclass
from typing import TypedDict

from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional


@dataclass(kw_only=True)
class Lifecycle:
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    prevent_destroy: VariableOrOptional[bool] = None
    """
    Whether to prevent the resource from being destroyed. If set to true, the resource will not be destroyed when running `databricks bundle destroy`.
    """

    @classmethod
    def from_dict(cls, value: "LifecycleDict") -> "Lifecycle":
        return _transform(cls, value)

    def as_dict(self) -> "LifecycleDict":
        return _transform_to_json_value(self)  # type:ignore


class LifecycleDict(TypedDict, total=False):
    """
    Lifecycle is a struct that contains the lifecycle settings for a resource. It controls the behavior of the resource when it is deployed or destroyed.
    """

    prevent_destroy: VariableOrOptional[bool]
    """
    Whether to prevent the resource from being destroyed. If set to true, the resource will not be destroyed when running `databricks bundle destroy`.
    """


LifecycleParam = LifecycleDict | Lifecycle
