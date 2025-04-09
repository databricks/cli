from dataclasses import dataclass, field
from typing import TYPE_CHECKING, Any, TypedDict

from databricks.bundles.core import Resource, VariableOrList, VariableOrOptional
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value

if TYPE_CHECKING:
    from typing_extensions import Self

# TODO generate Pipeline class from jsonschema


@dataclass(kw_only=True)
class Pipeline(Resource):
    """"""

    name: VariableOrOptional[str]

    # permission field is always present after normalization, add stub not to error on unknown property
    permissions: VariableOrList[Any] = field(default_factory=list)

    @classmethod
    def from_dict(cls, value: "PipelineDict") -> "Self":
        return _transform(cls, value)

    def as_dict(self) -> "PipelineDict":
        return _transform_to_json_value(self)  # type:ignore


class PipelineDict(TypedDict, total=False):
    """"""

    name: VariableOrOptional[str]
    """
    TODO
    """


PipelineParam = Pipeline | PipelineDict
