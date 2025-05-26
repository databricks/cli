import sys
import inspect
import importlib

from dataclasses import dataclass
from argparse import ArgumentParser
from typing import Callable, Generic, ParamSpec, TYPE_CHECKING

if TYPE_CHECKING:
    from databricks.bundles.jobs import PythonWheelTask

_P = ParamSpec("_P")


@dataclass
class PythonWheelTaskFunction(Generic[_P]):
    package_name: str
    entry_point: str
    func: Callable[_P, None]

    def __call__(self, *args: _P.args, **kwargs: _P.kwargs) -> "PythonWheelTask":
        from databricks.bundles.jobs import PythonWheelTask

        if args:
            raise ValueError("Only keyword arguments are supported")

        func_full_name = f"{self.func.__module__}:{self.func.__name__}"
        parameters: list = [func_full_name] + [f"--{k}={v}" for k, v in kwargs.items()]

        return PythonWheelTask(
            package_name=self.package_name,
            entry_point=self.entry_point,
            parameters=parameters,
        )


def python_wheel_task(func: Callable[_P, None]) -> Callable[_P, "PythonWheelTask"]:
    if "." in func.__qualname__:
        raise ValueError(
            "Only function at module top level can be used as a task entry point"
        )

    return PythonWheelTaskFunction[_P](
        # must match 'project.name' in pyproject.toml
        package_name="my_project",
        # entry_point must be defined in 'project.entry-points.packages' in pyproject.toml
        entry_point="python_wheel_task",
        func=func,
    )


def main():
    """
    Entry point for the Python wheel task.

    This function is referenced through entry_point and package_name parameters in the PythonWheelTask,
    and entry point is defined in 'project.entry-points.packages' in pyproject.toml.
    """

    func_full_name = sys.argv[1]
    func_module_name, func_name = func_full_name.split(":")

    func_module = importlib.import_module(func_module_name)
    decorated_func: PythonWheelTaskFunction = getattr(func_module, func_name)
    func = decorated_func.func

    parser = ArgumentParser()
    for param in inspect.signature(func).parameters.values():
        if param.annotation is not inspect.Parameter.empty:
            parser.add_argument(
                f"--{param.name}",
                type=param.annotation,
                required=param.default is inspect.Parameter.empty,
                help=f"Argument {param.name} of type {param.annotation}",
            )
        else:
            parser.add_argument(
                f"--{param.name}",
                type=str,
                required=param.default is inspect.Parameter.empty,
                help=f"Argument {param.name} of type str",
            )

    ns = parser.parse_args(sys.argv[2:])
    kwargs = vars(ns)

    func(**kwargs)
