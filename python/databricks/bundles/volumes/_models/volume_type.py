from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING
from enum import Enum
from dataclasses import dataclass, replace, field

from databricks.bundles.core._resource import Resource
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict

from databricks.bundles.volumes._models.privilege_assignment import PrivilegeAssignment, PrivilegeAssignmentDict, PrivilegeAssignmentParam
from databricks.bundles.volumes._models.lifecycle import Lifecycle, LifecycleDict, LifecycleParam
from databricks.bundles.volumes._models.volume import Volume, VolumeDict, VolumeParam
from databricks.bundles.volumes._models.privilege import Privilege, PrivilegeParam


class VolumeType(Enum):
    MANAGED = "MANAGED"
    EXTERNAL = "EXTERNAL"

VolumeTypeParam = Literal["MANAGED", "EXTERNAL"] | VolumeType
