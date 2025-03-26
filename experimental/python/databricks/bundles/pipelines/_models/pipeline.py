from dataclasses import dataclass
from typing import TYPE_CHECKING, TypedDict

from databricks.bundles.core import Resource, VariableOrOptional
from databricks.bundles.core._transform import _transform
from databricks.bundles.core._transform_to_json import _transform_to_json_value

if TYPE_CHECKING:
    from typing_extensions import Self

# TODO generate this class and its dependencies


@dataclass(kw_only=True)
class Pipeline(Resource):
    """"""

    name: VariableOrOptional[str]
    """
    TODO
    """

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
