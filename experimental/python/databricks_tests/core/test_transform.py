from dataclasses import dataclass
from enum import Enum
from typing import Optional

import pytest

from databricks.bundles.core import (
    Variable,
    VariableOr,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.core._transform import _transform


class Color(Enum):
    RED = "red"
    BLUE = "blue"


@dataclass
class MyDataclass:
    color: Optional[Color]


def test_transform_int():
    @dataclass
    class Fake:
        field: Optional[int] = None

    out = _transform(Fake, {"field": 42})

    assert out == Fake(field=42)


def test_transform_bool():
    @dataclass
    class Fake:
        field: Optional[bool] = None

    out = _transform(Fake, {"field": "false"})

    assert out == Fake(field=False)


def test_transform_str():
    @dataclass
    class Fake:
        field: Optional[str] = None

    out = _transform(Fake, {"field": "test"})

    assert out == Fake(field="test")


def test_transform_str_list():
    @dataclass
    class Fake:
        field: Optional[list[str]] = None

    out = _transform(Fake, {"field": ["a", "b"]})

    assert out == Fake(field=["a", "b"])


def test_transform_str_to_list():
    with pytest.raises(ValueError) as exc_info:
        _transform(list[str], "abc")

    assert str(exc_info.value) == "Unexpected type: list[str] for 'abc'"


def test_transform_str_to_dict():
    with pytest.raises(ValueError) as exc_info:
        _transform(dict[str, str], "abc")

    assert str(exc_info.value) == "Unexpected type: dict[str, str] for 'abc'"


def test_transform_none_to_optional_list():
    @dataclass
    class Fake:
        field: Optional[list[str]] = None

    out = _transform(Fake, {"field": None})

    assert out == Fake(field=None)


def test_transform_none_to_list():
    @dataclass
    class Fake:
        field: list[str]

    out = _transform(Fake, {"field": None})

    assert out == Fake(field=[])


def test_transform_none_to_optional_dict():
    @dataclass
    class Fake:
        field: Optional[dict[str, str]] = None

    out = _transform(Fake, {"field": None})

    assert out == Fake(field=None)


def test_transform_none_to_dict():
    @dataclass
    class Fake:
        field: dict[str, str]

    out = _transform(Fake, {"field": None})

    assert out == Fake(field={})


def test_transform_none_to_dict_of_int():
    @dataclass
    class Fake:
        field: dict[str, int]

    out = _transform(Fake, {"field": None})

    assert out == Fake(field={})


def test_transform_enum_from_str():
    @dataclass
    class Fake:
        field: Optional[Color] = None

    out = _transform(Fake, {"field": "red"})

    assert out == Fake(field=Color.RED)


def test_transform_enum_from_enum():
    @dataclass
    class Fake:
        field: Optional[Color] = None

    out = _transform(Fake, {"field": Color.RED})

    assert out == Fake(field=Color.RED)


def test_transform_enum_list():
    @dataclass
    class Fake:
        field: list[Color]

    out = _transform(Fake, {"field": ["red", "blue"]})

    assert out == Fake(field=[Color.RED, Color.BLUE])


def test_transform_nested_class_as_class():
    @dataclass
    class Nested:
        field: VariableOr[int]

    @dataclass
    class Fake:
        nested: Nested

    out = _transform(Fake, {"nested": Nested(field=42)})

    assert out == Fake(nested=Nested(field=42))


# these have to be defined in top-level, or types don't resolve
@dataclass
class ForwardRefA:
    b: Optional["ForwardRefB"]


@dataclass
class ForwardRefB:
    value: int


def test_transform_forward_ref():
    out = _transform(ForwardRefA, {"b": {"value": 42}})

    assert out == ForwardRefA(b=ForwardRefB(value=42))


def test_transform_dict_keys():
    @dataclass
    class Fake:
        tags: dict[str, VariableOr[str]]

    job = Fake(
        tags={
            "key1": Variable(path="var.my_var", type=str),  # type:ignore
        }
    )

    assert job.tags == {"key1": Variable(path="var.my_var", type=str)}


def test_unknown_fields():
    @dataclass
    class Fake:
        field: Optional[str] = None

    with pytest.raises(ValueError) as exc_info:
        _transform(Fake, {"field": "test", "unknown": "unknown"})

    assert str(exc_info.value) == "Unexpected field 'unknown' for class Fake"


def test_transform_cls_field():
    # we have a hack for handing "cls" field coming from locals()
    # test that we don't break normal case when it doesn't happen

    @dataclass
    class Fake:
        cls: str

    out = _transform(Fake, {"cls": "test"})

    assert out == Fake(cls="test")


def test_transform_none_to_variable_or_list():
    @dataclass
    class Fake:
        field: VariableOrList[str]

    out = _transform(Fake, {"field": None})

    assert out == Fake(field=[])


def test_forward_ref():
    @dataclass
    class A:
        field: VariableOrOptional["MyDataclass"]

    out = _transform(A, {"field": {"color": "red"}})

    assert out == A(field=MyDataclass(color=Color.RED))
