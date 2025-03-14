import re
import typing
from dataclasses import Field, fields, is_dataclass
from enum import Enum
from types import NoneType, UnionType
from typing import Any, ForwardRef, Optional, Type, TypeVar, Union, get_args, get_origin

from databricks.bundles.core._variable import (
    Variable,
    VariableOr,
    VariableOrDict,
    VariableOrList,
    VariableOrOptional,
)

__all__ = [
    "_transform",
]

_T = TypeVar("_T")


def _find_union_arg(value: Any, tpe: type) -> Optional[type]:
    """
    Returns the most appropriate type from the list to be used
    in _transform, or None if no type is suitable.
    """

    args = get_args(tpe)

    # If value not None, we support 5 forms:
    #
    # 1. VariableOrOptional[T] = Variable | T | None
    # 2. VariableOr[T]         = Variable | T
    # 3. Optional[T]           = T | None
    # 4. VariableOrList[T]     = VariableOr[list[T]]
    # 5. VariableOrDict[T]     = VariableOr[dict[str, T]]

    for arg in args:
        if _unwrap_variable(arg):
            if _unwrap_variable_path(value):
                return arg
        elif arg is type(None):
            if value is None:
                return type(None)
        else:
            # return the first non-variable and non-None type
            if value is not None:
                return arg

    # allow None to become empty "list" or "dict"
    if value is None:
        for arg in args:
            if _unwrap_list(arg):
                return arg

            if _unwrap_dict(arg):
                return arg

    return None


def _transform_variable_or_list(
    cls: Type[_T],
    value: VariableOrList[Any],
) -> VariableOrList[_T]:
    if isinstance(value, Variable):
        return value

    return [_transform_variable_or(cls, item) for item in value]


def _transform_variable_or_dict(
    cls: Type[_T],
    value: VariableOrDict[Any],
) -> VariableOrDict[_T]:
    if isinstance(value, Variable):
        return value

    return {key: _transform_variable_or(cls, item) for key, item in value.items()}


def _transform_variable_or(
    cls: Type[_T],
    value: VariableOr[Any],
) -> VariableOr[_T]:
    if isinstance(value, Variable):
        return Variable(path=value.path, type=cls)

    return _transform(cls, value)


def _transform_variable_or_optional(
    cls: Type[_T],
    value: VariableOrOptional[Any],
) -> VariableOrOptional[_T]:
    if isinstance(value, Variable):
        return Variable(path=value.path, type=cls)

    if value is None:
        return None

    return _transform(cls, value)


def _transform_field(cls: Type[_T], field: Field, value: Any) -> _T:
    # if we see strings like:
    #
    #   level: "PermissionLevel"
    #
    # we resolve the type for it

    if isinstance(field.type, ForwardRef) or isinstance(field.type, str):
        resolved = typing.get_type_hints(cls)
        return _transform(resolved[field.name], value)

    if origin := get_origin(field.type):
        if origin == Union or origin == UnionType:
            args = get_args(field.type)

            for arg in args:
                if isinstance(arg, (ForwardRef, str)):
                    resolved = typing.get_type_hints(cls)

                    return _transform(resolved[field.name], value)

    if arg := _unwrap_optional(field.type):
        if value is None:
            return None  # type: ignore
        if isinstance(arg, ForwardRef) or isinstance(arg, str):
            resolved = typing.get_type_hints(cls)
            return _transform(resolved[field.name], value)

    return _transform(field.type, value)


def _transform(cls: Type[_T], value: Any) -> _T:
    origin = get_origin(cls)

    if is_dataclass(cls) and isinstance(value, cls):  # type:ignore
        return value

    if isinstance(value, Enum) and isinstance(value, cls):
        return value  # type:ignore

    # if 'cls' is Variable[T], and 'value' is "${xxx}", return Variable(path='xxx', type=T)
    if arg := _unwrap_variable(cls):
        if path := _unwrap_variable_path(value):
            return Variable(path=path, type=arg)  # type:ignore

    if arg := _unwrap_optional(cls):
        if value is None:
            return None  # type:ignore

        return _transform(arg, value)
    elif origin == Union:
        if union_arg := _find_union_arg(value, cls):
            if union_arg is type(None):
                assert value is None

                return None  # type:ignore

            return _transform(union_arg, value)

        raise ValueError(f"Unexpected type: {cls} for {value}")
    elif is_dataclass(cls) and isinstance(value, dict):
        cls_fields = fields(cls)
        field_names = {field.name for field in cls_fields}

        # FIXME: we capture 'cls' in locals(), this should be fixed separately
        if value.get("cls") == cls:
            del value["cls"]

        for key in value:
            if key not in field_names:
                raise ValueError(f"Unexpected field '{key}' for class {cls.__name__}")

        kwargs = {
            field.name: _transform_field(cls, field, value[field.name])
            for field in cls_fields
            if field.name in value
        }

        return cls(**kwargs)  # type:ignore
    elif origin is list:
        if value is None:
            return []  # type:ignore
        [arg] = get_args(cls)

        if isinstance(value, str):
            raise ValueError(f"Unexpected type: {cls} for '{value}'")

        return [_transform(arg, item) for item in value]  # type:ignore
    elif cls is str:
        return str(value)  # type:ignore
    elif cls is int:
        return int(value)  # type:ignore
    elif cls is bool:
        if isinstance(value, bool):
            return value  # type:ignore

        if value in ["true", "false"]:
            return value == "true"  # type:ignore

        raise ValueError(f"Unexpected type: {cls} for '{value}'")
    elif issubclass(origin or cls, Enum):
        return cls(str(value))  # type:ignore
    elif cls == NoneType and value is None:
        return value
    elif arg := _unwrap_dict(cls):
        # allow to transform None to empty dict even if type doesn't allow None
        # this is used for "create" method that has different type signature than dataclass
        if value is None:
            return {}  # type:ignore

        [key_arg, value_arg] = arg

        assert key_arg is str

        if not isinstance(value, dict):
            raise ValueError(f"Unexpected type: {cls} for '{value}'")

        out = {key: _transform(value_arg, item) for key, item in value.items()}

        return out  # type:ignore
    else:
        raise ValueError(f"Unexpected type: {cls} for {value}")


def _unwrap_optional(tpe: type) -> Optional[type]:
    if origin := get_origin(tpe):
        if origin == Union or origin == UnionType:
            args = get_args(tpe)

            if len(args) == 2 and args[1] is type(None):
                return args[0]
            elif len(args) == 2 and args[0] is type(None):
                return args[1]

    return None


def _unwrap_list(tpe: type) -> Optional[type]:
    if origin := get_origin(tpe):
        if origin is list:
            args = get_args(tpe)

            if len(args) == 1:
                return args[0]

    return None


def _unwrap_dict(type: type) -> Optional[tuple[type, type]]:
    if origin := get_origin(type):
        if origin is dict:
            args = get_args(type)

            if len(args) == 2:
                return args[0], args[1]

    return None


def _unwrap_variable(tpe: type) -> Optional[type]:
    if origin := get_origin(tpe):
        if origin == Variable:
            return get_args(tpe)[0]

    return None


# Regex for string corresponding to variables.
#
# The source of truth is regex in libs/dyn/dynvar/ref.go
#
# Example:
#   - "${a.b}"
#   - "${a.b.c}"
#   - "${a.b[0].c}"
_base_var_def = r"[a-zA-Z]+([-_]*[a-zA-Z0-9]+)*"
_variable_regex = re.compile(
    r"\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}" % (_base_var_def, _base_var_def)
)


def _unwrap_variable_path(value: Any) -> Optional[str]:
    if isinstance(value, str):
        if match := _variable_regex.match(value):
            path = match.group(1)

            if match.start() == 0 and match.end() == len(value):
                return path
            else:
                return None

    # we ignore any generic arguments to Variable, because they
    # only exist for typechecking
    #
    # we should be able to extract path from any variable no
    # matter how it's typed, this makes any variable transformable
    # to any other variable

    if isinstance(value, Variable):
        return value.path

    return None


def _display_type(tpe: type):
    if not get_args(tpe):
        return tpe.__name__

    return repr(tpe)
