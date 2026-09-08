from dataclasses import dataclass
from typing import Callable

from databricks.bundles.core._resource import Resource


@dataclass(kw_only=True)
class TestCase:
    __test__ = False  # not a pytest test class despite the name

    add_resource: Callable
    dict_example: dict
    dataclass_example: Resource
    mutator: Callable
