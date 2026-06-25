from dataclasses import dataclass, field
from enum import Enum
from typing import Optional, Union

import pytest

from databricks.bundles.core import (
    Variable,
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)
from databricks.bundles.core._transform import (
    _find_union_arg,
    _transform,
    _unwrap_variable_path,
)


class FakeEnum(Enum):
    VALUE1 = "value1"
    VALUE2 = "value2"


@dataclass
class Fake:
    dict_field: VariableOrOptional[dict[str, VariableOr[str]]] = field(
        default_factory=dict
    )
    str_field: VariableOrOptional[str] = None
    enum_field: VariableOrOptional[FakeEnum] = None


@pytest.mark.parametrize(
    "input,tpe,expected",
    [
        (
            {"key1": "${var.my_var}"},
            dict[str, VariableOr[str]],
            {"key1": Variable(path="var.my_var", type=str)},
        ),
        (
            {"dict_field": {"key1": "${var.my_var}"}},
            Fake,
            Fake(dict_field={"key1": Variable(path="var.my_var", type=str)}),
        ),
        (
            {"dict_field": "${var.my_var}"},
            Fake,
            Fake(
                dict_field=Variable(
                    path="var.my_var",
                    type=dict[str, VariableOr[str]],
                )
            ),
        ),
        (
            {"str_field": "${var.my_var}"},
            Fake,
            Fake(str_field=Variable(path="var.my_var", type=str)),
        ),
        (
            "${var.my_var}",
            Variable[str],
            Variable(path="var.my_var", type=str),
        ),
        (
            "${var.my_var}",
            Variable[dict[str, str]],
            Variable(path="var.my_var", type=dict[str, str]),
        ),
        (
            "${var.my_var}",
            Variable[Fake],
            Variable(path="var.my_var", type=Fake),
        ),
        (
            "${var.my_var}",
            Union[Variable[Fake], Fake],
            Variable(path="var.my_var", type=Fake),
        ),
        # _transform keeps variable types when possible
        (
            {"enum_field": Variable(path="var.my_var", type=FakeEnum)},
            Fake,
            Fake(enum_field=Variable(path="var.my_var", type=FakeEnum)),
        ),
        # _transform can get a Variable[str], that should be turned into Variable[Enum]
        (
            {"enum_field": Variable(path="var.my_var", type=str)},
            Fake,
            Fake(enum_field=Variable(path="var.my_var", type=FakeEnum)),
        ),
        (
            {"enum_field": "value1"},
            Fake,
            Fake(enum_field=FakeEnum.VALUE1),
        ),
    ],
)
def test_transform_variable(input, tpe, expected):
    assert _transform(tpe, input) == expected


@pytest.mark.parametrize(
    "input,tpe,expected",
    # The following lines use `type: ignore` because Pyright raises an error: `__getitem__` method is not defined on the type `UnionType`.
    # This is a known issue, and more details can be found at: https://github.com/microsoft/pyright/issues/8319
    [
        # if value looks like variable, it should be variable
        (
            "${var.my_var}",
            VariableOr[bool],  # type:ignore
            Variable[bool],
        ),
        (
            "${var.my_var}",
            VariableOrOptional[bool],  # type:ignore
            Variable[bool],
        ),
        # if not, we should return value, even if type doesn't match, because
        # it's our best guess
        (
            42,
            VariableOr[str],  # type:ignore
            str,
        ),
        (
            42,
            VariableOrOptional[str],  # type:ignore
            str,
        ),
        # variable types only matter for typechecker, we ignore them
        # when we do instanceof check
        (
            Variable(path="my_var", type=str),
            VariableOr[FakeEnum],  # type:ignore
            Variable[FakeEnum],
        ),
        (
            Variable(path="my_var", type=str),
            VariableOrOptional[FakeEnum],  # type:ignore
            Variable[FakeEnum],
        ),
        # if value is None, we should always choose NoneType
        (
            None,
            VariableOrOptional[str],  # type:ignore
            type(None),
        ),
        (
            None,
            Optional[str],
            type(None),
        ),
        # if value is None, but value is non-nullable we need to return None value
        (
            None,
            VariableOr[str],  # type:ignore
            None,
        ),
        (
            [],
            VariableOrList[int],
            list[VariableOr[int]],
        ),
        (
            {},
            VariableOrDict[int],
            dict[str, VariableOr[int]],
        ),
        # when we see "None", it can become list or dict even if type is not optional
        # this is needed for "create" method that always has optional collections
        # while dataclasses have them required
        (
            None,
            VariableOrList[int],
            list[VariableOr[int]],
        ),
        (
            None,
            VariableOrDict[int],
            dict[str, VariableOr[int]],
        ),
    ],
)
def test_find_union_arg(input, tpe, expected):
    assert _find_union_arg(input, tpe) == expected


@pytest.mark.parametrize(
    "input,expected",
    [
        pytest.param(
            "${var.my_var}",
            "var.my_var",
            id="simple variable",
        ),
        pytest.param(
            "${var.my_var} ${var.my_var}",
            None,
            id="multiple variables aren't allowed",
        ),
        pytest.param(
            "${var.my_var[0]}",
            "var.my_var[0]",
            id="variable with subscript",
        ),
        pytest.param(
            "${var.my_var[0].foo}",
            "var.my_var[0].foo",
            id="variable with subscript + attribute",
        ),
        pytest.param(
            "${var.my_var[0].foo.bar}",
            "var.my_var[0].foo.bar",
            id="variable with multiple subscripts",
        ),
        pytest.param(
            "${var.my_var[0].foo[0]}",
            "var.my_var[0].foo[0]",
            id="variable with subscript inside attribute",
        ),
    ],
)
def test_unwrap_variable_path(input, expected):
    assert _unwrap_variable_path(input) == expected
