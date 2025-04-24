import ast
import importlib
import importlib.util
import inspect
import pkgutil
from textwrap import dedent
from types import ModuleType
from typing import Any, Iterable

from databricks.bundles.core._diagnostics import Diagnostics
from databricks.bundles.core._location import Location
from databricks.bundles.core._resource import Resource
from databricks.bundles.core._resources import Resources

__all__ = [
    "load_resources_from_current_package_module",
    "load_resources_from_package_module",
    "load_resources_from_modules",
    "load_resources_from_module",
]

"""
Functions to load resources from modules.

Each resource is a variable that is an instance of `Resource` class. Python variable names are used as resource names.
Resource names for a given resource type must be unique within a bundle.
"""


def load_resources_from_current_package_module() -> Resources:
    """
    Load resources from all submodules of the current package module.
    """

    stack = inspect.stack()
    frame = stack[1].frame
    module = inspect.getmodule(frame)

    if not module:
        raise RuntimeError("Failed to get current module from stack frame")

    return load_resources_from_package_module(module)


def load_resources_from_package_module(
    package_module: ModuleType,
) -> Resources:
    """
    Load resources from all submodules of the given package module.

    :param package_module: package module to load resources from
    """

    package_spec = package_module.__spec__
    if not package_spec:
        raise ValueError(f"Module '{package_module.__name__}' has no __spec__")

    module_names = []
    if package_spec.origin:
        module_names.append(package_spec.name)

    if package_spec.submodule_search_locations:
        package_paths = package_spec.submodule_search_locations

        prefix = package_spec.name + "."

        for loader, module_name, is_pkg in pkgutil.walk_packages(package_paths, prefix):
            module_names.append(module_name)

    module_names.sort()  # create deterministic order
    modules = []
    import_diagnostics = Diagnostics()

    for module_name in module_names:
        try:
            modules.append(importlib.import_module(module_name))
        except Exception as exc:
            import_diagnostics = import_diagnostics.extend(
                Diagnostics.from_exception(
                    exc,
                    summary=f"Error while loading '{module_name}'",
                )
            )

    resources = load_resources_from_modules(modules)
    resources.add_diagnostics(import_diagnostics)

    return resources


def load_resources_from_modules(modules: Iterable[ModuleType]) -> Resources:
    """
    Load resources from the given modules.

    For recursive loading of resources from submodules, use `load_resources_from_package_module`.

    :param modules: list of modules to load resources from
    """

    resources = Resources()

    for module in modules:
        resources.add_resources(load_resources_from_module(module))

    return resources


def load_resources_from_module(module: ModuleType) -> Resources:
    """
    Load resources from the given module.

    For recursive loading of resources from submodules, use `load_resources_from_package_module`.

    :param module: module to load resources from
    """

    resources = Resources()

    try:
        locations = _parse_locations(module)
        fallback_location = Location(file=module.__file__) if module.__file__ else None

        for member_name, member in inspect.getmembers(module, _is_resource):
            location = locations.get(member_name, fallback_location)

            resources.add_resource(member_name, member, location=location)
    except Exception as exc:
        resources.add_diagnostics(
            Diagnostics.from_exception(
                exc,
                summary=f"Error while loading '{module.__name__}'",
            )
        )

    return resources


def _parse_locations(module: ModuleType) -> dict[str, Location]:
    locations = {}
    file = getattr(module, "__file__", "<unknown>")

    try:
        source = inspect.getsource(module)
    except OSError:
        # failed to get source code, non-fatal error because locations are only cosmetic
        return locations

    node = ast.parse(dedent(source), filename=file)

    assert isinstance(node, ast.Module)

    for stmt in node.body:
        if not isinstance(stmt, ast.Assign):
            continue

        for target in stmt.targets:
            if not isinstance(target, ast.Name):
                continue

            var_name = target.id

            locations[var_name] = Location(
                line=stmt.lineno,
                # do conversion: col_offset is 0-based, and column is 1-based
                column=stmt.col_offset + 1,
                file=file,
            )

    return locations


def _is_resource(member: Any) -> bool:
    return isinstance(member, Resource)
