from dataclasses import dataclass
from typing import Callable

from databricks.bundles.core._resource import Resource


@dataclass(kw_only=True)
class TestCase:
    add_resource: Callable
    dict_example: dict
    dataclass_example: Resource
    mutator: Callable
