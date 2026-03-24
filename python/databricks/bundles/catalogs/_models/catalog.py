from dataclasses import dataclass, field
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.catalogs._models.catalog_isolation_mode import (
    CatalogIsolationMode,
    CatalogIsolationModeParam,
)
from databricks.bundles.catalogs._models.enable_predictive_optimization import (
    EnablePredictiveOptimization,
    EnablePredictiveOptimizationParam,
)
from databricks.bundles.catalogs._models.lifecycle import Lifecycle, LifecycleParam
from databricks.bundles.catalogs._models.privilege_assignment import (
    PrivilegeAssignment,
    PrivilegeAssignmentParam,
)
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import (
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class Catalog(Resource):
    """"""

    name: VariableOr[str]

    comment: VariableOrOptional[str] = None

    connection_name: VariableOrOptional[str] = None

    enable_predictive_optimization: VariableOrOptional[EnablePredictiveOptimization] = (
        None
    )

    grants: VariableOrList[PrivilegeAssignment] = field(default_factory=list)

    isolation_mode: VariableOrOptional[CatalogIsolationMode] = None

    lifecycle: VariableOrOptional[Lifecycle] = None

    options: VariableOrDict[str] = field(default_factory=dict)

    owner: VariableOrOptional[str] = None

    properties: VariableOrDict[str] = field(default_factory=dict)

    provider_name: VariableOrOptional[str] = None

    share_name: VariableOrOptional[str] = None

    storage_root: VariableOrOptional[str] = None

    @classmethod
    def from_dict(cls, value: "CatalogDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "CatalogDict":
        return _transform_to_json_value(self)  # type:ignore


class CatalogDict(TypedDict, total=False):
    """"""

    name: VariableOr[str]

    comment: VariableOrOptional[str]

    connection_name: VariableOrOptional[str]

    enable_predictive_optimization: VariableOrOptional[
        EnablePredictiveOptimizationParam
    ]

    grants: VariableOrList[PrivilegeAssignmentParam]

    isolation_mode: VariableOrOptional[CatalogIsolationModeParam]

    lifecycle: VariableOrOptional[LifecycleParam]

    options: VariableOrDict[str]

    owner: VariableOrOptional[str]

    properties: VariableOrDict[str]

    provider_name: VariableOrOptional[str]

    share_name: VariableOrOptional[str]

    storage_root: VariableOrOptional[str]


CatalogParam = CatalogDict | Catalog
