from dataclasses import fields, is_dataclass
from enum import Enum
from typing import Any, Union

from databricks.bundles.core._variable import Variable

__all__ = [
    "_transform_to_json_value",
]


def _transform_to_json_value(
    value: Any,
) -> Union[str, bool, int, float, dict[str, Any], list[Any], None]:
    # you should read the returned datatype as recursive JSON

    if isinstance(value, Enum):
        return value.value
    elif isinstance(value, Variable):
        # order matters, because Variable is a dataclass as well
        return value.value
    elif is_dataclass(value):
        return _transform_to_json_object(value)
    elif isinstance(value, list):
        return [_transform_to_json_value(el) for el in value]
    elif isinstance(value, dict):
        return {
            item_key: _transform_to_json_value(item_value)
            for item_key, item_value in value.items()
        }
    elif isinstance(value, Union[str, bool, int, float, None]):
        return value

    raise ValueError(f"Can't serialize '{value}'")


def _transform_to_json_object(value: Any) -> dict[str, Any]:
    # assert is_dataclass(cls) doesn't always work, see:
    # https://github.com/python/cpython/issues/92893

    assert is_dataclass(value), f"Expected dataclass but got {value}"

    out = {}

    for field_name, _ in _stable_fields(value):
        field_value = _transform_to_json_value(getattr(value, field_name))

        # we implement omittempty semantics here skipping None, [], and {}
        if field_value not in [None, [], {}]:
            out[field_name] = field_value

    return out


def _stable_fields(value: Any) -> list[tuple[str, type | str]]:
    assert is_dataclass(value)
    assert not isinstance(value, type)

    field_names = [(field.name, field.type) for field in fields(value)]

    field_names.sort(key=lambda x: x[0])

    return field_names
