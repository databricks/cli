import argparse
import importlib
import inspect
import json
import logging
import os.path
import sys
from copy import deepcopy
from dataclasses import dataclass, field, fields, replace
from typing import (
    Callable,
    Optional,
    TextIO,
)

from databricks.bundles.core import Bundle, Diagnostics, Location, Resources
from databricks.bundles.core._resource_mutator import ResourceMutator
from databricks.bundles.core._resource_type import _ResourceType
from databricks.bundles.core._transform import _transform

__all__ = []


@dataclass
class _Args:
    phase: str
    input: str
    output: str
    diagnostics: str
    locations: Optional[str]  # FIXME should be required
    unknown_args: list[str]


@dataclass
class _Conf:
    resources: list[str] = field(default_factory=list)
    mutators: list[str] = field(default_factory=list)
    venv_path: Optional[str] = None

    @classmethod
    def from_dict(cls, d: dict) -> "_Conf":
        known_keys = [f.name for f in fields(cls)]
        unknown_keys = d.keys() - known_keys

        if unknown_keys:
            logging.warning(f"Unknown configuration keys: {unknown_keys}")

        return _transform(cls, {k: v for k, v in d.items() if k in known_keys})


def _parse_args(args: list[str]) -> _Args:
    parser = argparse.ArgumentParser()
    parser.add_argument("--phase", default=None)
    parser.add_argument("--input", default=None)
    parser.add_argument("--output", default=None)
    parser.add_argument("--diagnostics", default=None)
    parser.add_argument("--locations", default=None)

    parsed, unknown_args = parser.parse_known_args(args)

    if not parsed.phase:
        raise ValueError("Missing required argument --phase")

    if not parsed.input:
        raise ValueError("Missing required argument --input")

    if not parsed.output:
        raise ValueError("Missing required argument --output")

    if not parsed.diagnostics:
        raise ValueError("Missing required argument --diagnostics")

    return _Args(
        phase=parsed.phase,
        input=parsed.input,
        output=parsed.output,
        diagnostics=parsed.diagnostics,
        locations=parsed.locations,
        unknown_args=unknown_args,
    )


def _load_resources_from_input(input: dict) -> tuple[Resources, Diagnostics]:
    resources = Resources()
    diagnostics = Diagnostics()

    input_resources = input.get("resources", {})

    for tpe in _ResourceType.all():
        input_resources_by_tpe = input_resources.get(tpe.plural_name, {})

        for resource_name, resource_dict in input_resources_by_tpe.items():
            try:
                resource = _transform(tpe.resource_type, resource_dict)

                resources.add_resource(resource_name=resource_name, resource=resource)
            except Exception as exc:
                diagnostics = diagnostics.extend(
                    Diagnostics.from_exception(
                        exc=exc,
                        summary=f"Error while loading {tpe.singular_name}",
                        path=("resources", tpe.plural_name, resource_name),
                    )
                )

    return resources, diagnostics


def _apply_mutators(
    bundle: Bundle,
    resources: Resources,
    mutator_functions: list[ResourceMutator],
) -> tuple[Resources, Diagnostics]:
    diagnostics = Diagnostics()

    for tpe in _ResourceType.all():
        resources, diagnostics = diagnostics.extend_tuple(
            _apply_mutators_for_type(bundle, resources, tpe, mutator_functions)
        )

    return resources, diagnostics


def _apply_mutators_for_type(
    bundle: Bundle,
    resources: Resources,
    tpe: _ResourceType,
    mutator_functions: list[ResourceMutator],
) -> tuple[Resources, Diagnostics]:
    resources_dict = getattr(resources, tpe.plural_name)

    for resource_name, resource in resources_dict.items():
        for mutator in mutator_functions:
            if mutator.resource_type != tpe.resource_type:
                continue

            location = Location.from_callable(mutator.function)

            try:
                if _get_num_args(mutator.function) == 1:
                    new_resource = mutator.function(resource)
                else:
                    # defensive copy so that one function doesn't affect another
                    new_resource = mutator.function(deepcopy(bundle), resource)

                # mutating resource in-place works, but we can't tell when it happens,
                # so we only update location if new instance is returned

                if new_resource is not resource:
                    if location:
                        resources.add_location(
                            ("resources", tpe.plural_name, resource_name), location
                        )
                    resources_dict[resource_name] = new_resource
                    resource = new_resource
            except Exception as exc:
                mutator_name = mutator.function.__name__

                return resources, Diagnostics.from_exception(
                    exc=exc,
                    summary=f"Failed to apply '{mutator_name}' mutator",
                    location=location,
                    path=("resources", tpe.plural_name, resource_name),
                )

    return resources, Diagnostics()


def python_mutator(
    args: _Args,
) -> tuple[dict, dict[tuple[str, ...], Location], Diagnostics]:
    input = json.load(open(args.input))
    experimental = input.get("experimental", {})

    if experimental.get("pydabs", {}) != {}:
        return (
            {},
            {},
            Diagnostics.create_error(
                "'experimental/pydabs' is not supported by 'databricks-bundles', use 'experimental/python' instead",
                detail="",
                location=None,
                path=("experimental", "pydabs"),
            ),
        )

    conf_dict = experimental.get("python", {})
    conf = _transform(_Conf, conf_dict)
    bundle = _parse_bundle_info(input)

    if args.phase == "load_resources":
        resource_functions, diagnostics = _load_functions(conf.resources)
        if diagnostics.has_error():
            return input, {}, diagnostics

        resources, diagnostics = diagnostics.extend_tuple(
            _load_resources(bundle, resource_functions)
        )
        if diagnostics.has_error():
            return input, {}, diagnostics

        new_bundle = _append_resources(input, resources)
        locations = _relativize_locations(resources._locations)

        return new_bundle, locations, diagnostics.extend(resources.diagnostics)
    elif args.phase == "apply_mutators":
        mutator_functions, diagnostics = _load_resource_mutators(conf.mutators)
        if diagnostics.has_error():
            return input, {}, diagnostics

        resources, diagnostics = _load_resources_from_input(input)
        if diagnostics.has_error():
            return input, {}, diagnostics

        resources, diagnostics = diagnostics.extend_tuple(
            _apply_mutators(bundle, resources, mutator_functions)
        )
        if diagnostics.has_error():
            return input, {}, diagnostics

        new_bundle = _append_resources(input, resources)
        locations = _relativize_locations(resources._locations)

        return new_bundle, locations, diagnostics.extend(resources.diagnostics)
    else:
        return input, {}, Diagnostics.create_error(f"Unknown phase: {args.phase}")


def _parse_bundle_info(input: dict) -> Bundle:
    bundle = input.get("bundle", {})
    variables = {k: v.get("value") for k, v in input.get("variables", {}).items()}

    return Bundle(
        target=bundle["target"],
        variables=variables,
    )


def _append_resources(bundle: dict, resources: Resources) -> dict:
    """
    Append resources to input without modifying resources that are already present.
    """

    new_bundle = bundle.copy()

    for tpe in _ResourceType.all():
        resources_dict = getattr(resources, tpe.plural_name)

        if resources_dict:
            new_bundle["resources"] = new_bundle.get("resources", {})
            new_bundle["resources"][tpe.plural_name] = new_bundle["resources"].get(
                tpe.plural_name, {}
            )

            for resource_name, resource in resources_dict.items():
                new_bundle["resources"][tpe.plural_name][resource_name] = (
                    resource.as_dict()
                )

    return new_bundle


def _get_num_args(func: Callable) -> int:
    return len(inspect.signature(func).parameters)


def _load_resources(
    bundle: Bundle,
    functions: list[Callable],
) -> tuple[Resources, Diagnostics]:
    diagnostics = Diagnostics()
    resources = Resources()

    for function in functions:
        try:
            function_resources, diagnostics = diagnostics.extend_tuple(
                _load_resources_from_function(bundle, function)
            )

            resources.add_resources(function_resources)
        except Exception as exc:
            diagnostics = diagnostics.extend(
                Diagnostics.from_exception(
                    exc=exc,
                    summary="Failed to load resources",
                    location=Location.from_callable(function),
                )
            )

    return resources, diagnostics


def _load_functions(names: list[str]) -> tuple[list[Callable], Diagnostics]:
    diagnostics = Diagnostics()
    functions = []

    for name in names:
        try:
            function, diagnostics = diagnostics.extend_tuple(_load_function(name))

            if function:
                functions.append(function)
        except Exception as exc:
            diagnostics = diagnostics.extend(
                Diagnostics.from_exception(
                    exc=exc,
                    summary=f"Failed to load function '{name}'",
                )
            )

    return functions, diagnostics


def _load_resource_mutators(
    names: list[str],
) -> tuple[list[ResourceMutator], Diagnostics]:
    diagnostics = Diagnostics()
    functions = []

    for name in names:
        try:
            function, diagnostics = diagnostics.extend_tuple(
                _load_resource_mutator(name)
            )

            if function:
                functions.append(function)
        except Exception as exc:
            diagnostics = diagnostics.extend(
                Diagnostics.from_exception(
                    exc=exc,
                    summary=f"Failed to load mutator '{name}'",
                )
            )

    return functions, diagnostics


def _load_object(qualified_name: str) -> tuple[Optional[Callable], Diagnostics]:
    diagnostics = Diagnostics()
    [module_name, name] = qualified_name.split(":")

    common_error = qualified_name == "resources:load_resources"

    try:
        module = importlib.import_module(module_name)
    except Exception as exc:
        if isinstance(exc, ModuleNotFoundError) and exc.name == "resources":
            if common_error:
                return None, _explain_common_import_error(exc)

        return None, Diagnostics.from_exception(
            exc=exc,
            summary=f"Failed to import module '{module_name}'",
        )

    try:
        return getattr(module, name), diagnostics
    except AttributeError as exc:
        if common_error:
            return None, _explain_common_import_error(exc)

        return None, Diagnostics.from_exception(
            exc=exc,
            summary=f"Name '{name}' not found in module '{module_name}'",
            location=Location(file=module.__file__) if module.__file__ else None,
        )
    except Exception as exc:
        return None, Diagnostics.from_exception(
            exc=exc,
            summary=f"Failed to load function '{name}' from module '{module_name}'",
        )


def _explain_common_import_error(exc: Exception) -> Diagnostics:
    # a common case when default name of the module is not found
    # we can give a hint to the user how to fix it
    explanation = """Make sure to create a new Python file at resources/__init__.py with contents:

from databricks.bundles.core import load_resources_from_current_package_module


def load_resources():
    return load_resources_from_current_package_module()
"""
    return Diagnostics.from_exception(
        exc=exc,
        summary="Can't find function 'load_resources' in module 'resources'",
        explanation=explanation,
    )


def _load_function(qualified_name: str) -> tuple[Optional[Callable], Diagnostics]:
    [instance, diagnostics] = _load_object(qualified_name)
    [module_name, name] = qualified_name.split(":")

    if diagnostics.has_error():
        return None, diagnostics

    if instance and not isinstance(instance, Callable):
        return None, Diagnostics.create_error(
            f"Function '{name}' in module '{module_name}' is not callable",
        )

    return instance, diagnostics


def _load_resource_mutator(
    qualified_name: str,
) -> tuple[Optional[ResourceMutator], Diagnostics]:
    [instance, diagnostics] = _load_object(qualified_name)
    [module_name, name] = qualified_name.split(":")

    if diagnostics.has_error():
        return None, diagnostics

    if instance and not isinstance(instance, ResourceMutator):
        return None, Diagnostics.create_error(
            f"'{name}' in module '{module_name}' is not instance of ResourceMutator, did you decorate it with @<resource_type>_mutator?",
        )

    return instance, diagnostics


def _load_resources_from_function(
    bundle: Bundle,
    function: Callable,
) -> tuple[Resources, Diagnostics]:
    if _get_num_args(function) == 0:
        resources = function()
    else:
        # defensive copy so that one function doesn't affect another
        resources = function(deepcopy(bundle))

    assert isinstance(resources, Resources)

    return resources, Diagnostics()


def main(argv: list[str]) -> int:
    args = _parse_args(argv[1:])

    if args.unknown_args:
        logging.warning(f"Unknown arguments: {args.unknown_args}")

    logging.basicConfig(level=logging.DEBUG, stream=sys.stderr)
    new_bundle, locations, diagnostics = python_mutator(args)

    with open(args.diagnostics, "w") as f:
        _write_diagnostics(f, diagnostics)

    if locations_path := args.locations:
        with open(locations_path, "w") as f:
            _write_locations(f, locations)

    with open(args.output, "w") as f:
        _write_output(f, new_bundle)

    return 1 if diagnostics.has_error() else 0


def _write_diagnostics(f: TextIO, diagnostics: Diagnostics) -> None:
    for diagnostic in diagnostics.items:
        obj = diagnostic.as_dict()

        if obj.get("path"):
            obj["path"] = ".".join(obj["path"])

        json.dump(obj, f)
        f.write("\n")


def _write_output(f: TextIO, bundle: dict) -> None:
    json.dump(bundle, f)


def _relativize_locations(
    locations: dict[tuple[str, ...], Location],
) -> dict[tuple[str, ...], Location]:
    return {
        path: _relativize_location(location) for path, location in locations.items()
    }


def _relativize_location(location: Location) -> Location:
    return replace(location, file=_relativize_path(location.file))


def _relativize_path(path: str) -> str:
    if not os.path.isabs(path):
        return path

    cwd = os.getcwd()
    common = os.path.commonpath([os.getcwd(), path])

    if common == cwd:
        return os.path.relpath(path, cwd)

    return path


def _write_locations(f: TextIO, locations: dict[tuple[str, ...], Location]) -> None:
    for path, location in locations.items():
        obj = {"path": ".".join(path), **location.as_dict()}

        json.dump(obj, f)
        f.write("\n")


if __name__ == "__main__":
    exit_code = main(sys.argv)

    sys.exit(exit_code)
