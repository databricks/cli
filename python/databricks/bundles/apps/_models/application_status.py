from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.apps._models.application_state import (
    ApplicationState,
    ApplicationStateParam,
)
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value
from databricks.bundles.core._variable import VariableOrOptional

if TYPE_CHECKING:
    from typing_extensions import Self


@dataclass(kw_only=True)
class ApplicationStatus:
    """"""

    message: VariableOrOptional[str] = None

    state: VariableOrOptional[ApplicationState] = None

    @classmethod
    def from_dict(cls, value: "ApplicationStatusDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "ApplicationStatusDict":
        return _transform_to_json_value(self)  # type:ignore


class ApplicationStatusDict(TypedDict, total=False):
    """"""

    message: VariableOrOptional[str]

    state: VariableOrOptional[ApplicationStateParam]


ApplicationStatusParam = ApplicationStatusDict | ApplicationStatus
