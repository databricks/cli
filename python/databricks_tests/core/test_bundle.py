from dataclasses import dataclass

import pytest

from databricks.bundles.core import (
    Bundle,
    Variable,
    VariableOr,
    VariableOrList,
)


def test_variable_not_a_variable():
    bundle = Bundle(target="development", variables={})

    assert bundle.resolve_variable(42) == 42


def test_variable_unknown():
    bundle = Bundle(target="development", variables={"bar": 1})
    var = Variable(path="var.foo", type=str)

    with pytest.raises(ValueError) as exc_info:
        bundle.resolve_variable(var)

    assert (
        str(exc_info.value)
        == "Can't find 'foo' variable. Did you define it in databricks.yml?"
    )


def test_variable_get_bool_true():
    bundle = Bundle(target="development", variables={"foo": "true"})
    var = Variable(path="var.foo", type=bool)

    assert bundle.resolve_variable(var) is True


def test_variable_get_bool_false():
    bundle = Bundle(target="development", variables={"foo": "false"})
    var = Variable(path="var.foo", type=bool)

    assert bundle.resolve_variable(var) is False


def test_variable_complex():
    @dataclass
    class Fake:
        name: str

    bundle = Bundle(target="development", variables={"foo": {"name": "bar"}})
    var = Variable(path="var.foo", type=Fake)

    assert bundle.resolve_variable(var) == Fake(name="bar")


def test_variable_failed_to_parse():
    @dataclass
    class Fake:
        name: str

    bundle = Bundle(target="development", variables={"foo": {}})
    var = Variable(path="var.foo", type=Fake)

    with pytest.raises(ValueError) as exc_info:
        bundle.resolve_variable(var)

    assert str(exc_info.value) == "Failed to read 'foo' variable value"
    assert exc_info.value.__cause__


def test_variable_uses_variable():
    bundle = Bundle(target="development", variables={"foo": "${var.bar}"})
    var = Variable(path="var.foo", type=Variable[str])

    assert bundle.resolve_variable(var) == Variable(path="var.bar", type=str)


def test_variable_uses_variable_not_supported():
    bundle = Bundle(target="development", variables={"foo": "${var.bar}"})
    var = Variable(path="var.foo", type=str)

    with pytest.raises(ValueError) as exc_info:
        bundle.resolve_variable(var)

    assert (
        str(exc_info.value)
        == "Failed to resolve 'foo' because refers to another variable 'var.bar'. "
        "Change variable type to Variable[VariableOr[str]]"
    )


def test_variable_uses_variable_not_supported_complex():
    @dataclass
    class Fake:
        name: str

    bundle = Bundle(target="development", variables={"foo": "${var.bar}"})
    var = Variable(path="var.foo", type=Fake)

    with pytest.raises(ValueError) as exc_info:
        bundle.resolve_variable(var)

    assert (
        str(exc_info.value)
        == "Failed to resolve 'foo' because refers to another variable 'var.bar'. "
        "Change variable type to Variable[VariableOr[Fake]]"
    )


def test_resolve_variable_or_list():
    bundle = Bundle(
        target="development", variables={"foo": ["${var.bar}"], "bar": "baz"}
    )
    var: VariableOrList[str] = Variable(path="var.foo", type=list[VariableOr[str]])

    assert bundle.resolve_variable_list(var) == ["baz"]
