from dataclasses import dataclass, field
from enum import Enum
from typing import Optional

import pytest

from databricks.bundles.core import Variable
from databricks.bundles.core._transform_to_json import _transform_to_json_value


class Color(Enum):
    RED = "red"
    BLUE = "blue"


@dataclass
class Fake:
    enum_field: Optional[Color] = None
    int_field: Optional[int] = None
    list_field: list[int] = field(default_factory=list)
    dict_field: dict[str, int] = field(default_factory=dict)


@pytest.mark.parametrize(
    ("input", "expected"),
    [
        (
            [1, 2, 3],
            [1, 2, 3],
        ),
        (
            [],
            [],
        ),
        (
            # empty lists, dicts and None is omitted
            Fake(),
            {},
        ),
        (
            [{"a": Fake(enum_field=Color.RED)}],
            [{"a": {"enum_field": "red"}}],
        ),
        (
            [Fake(enum_field=Color.RED)],
            [{"enum_field": "red"}],
        ),
        (
            Variable(path="var.my_var", type=int),
            "${var.my_var}",
        ),
    ],
)
def test_transform_to_json_value(input, expected):
    assert _transform_to_json_value(input) == expected
