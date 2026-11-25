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

# Same strings as normalizeBool in libs/dyn/convert/normalize.go.
# See https://yaml.org/type/bool.html.
_TRUE_STRINGS = [
    "true",
    "True",
    "TRUE",
    "y",
    "Y",
    "yes",
    "Yes",
    "YES",
    "on",
    "On",
    "ON",
]
_FALSE_STRINGS = [
    "false",
    "False",
    "FALSE",
    "n",
    "N",
    "no",
    "No",
    "NO",
    "off",
    "Off",
    "OFF",
]


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

    # interface{} fields are generated as `Any`: there is no schema to normalize
    # against, so pass the value through unchanged. The bundle config mutators
    # validate it (string or map) on the Go side.
    if cls is Any:
        return value  # type:ignore

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
    elif cls is float:
        return float(value)  # type:ignore
    elif cls is bool:
        if isinstance(value, bool):
            return value  # type:ignore

        if value in _TRUE_STRINGS:
            return True  # type:ignore
        if value in _FALSE_STRINGS:
            return False  # type:ignore

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
# The source of truth is regex in libs/dyn/dynvar/ref.go.
# Behavioral parity is enforced by libs/dyn/dynvar/testdata/reference_vectors.json.
#
# The (?<!\$) lookbehind skips references escaped by a preceding "$": "$${a.b}" is a
# literal "${a.b}" for the Databricks runtime, not a bundle reference.
#
# Example:
#   - "${a.b}"
#   - "${a.b.c}"
#   - "${a.b[0].c}"
#
# Go's \p{N} also takes numbers that are not decimal digits (Unicode categories No
# and Nl, such as "²" or "Ⅻ"). Python's re has no class for them: \d leaves them out
# and \w takes them in, so [^\W\d_] alone would take "²" for a letter.
# Generated from Go's unicode.No and unicode.Nl tables (Unicode 17.0).
_non_decimal_numbers = (
    "\u00b2\u00b3\u00b9\u00bc-\u00be\u09f4-\u09f9\u0b72-\u0b77\u0bf0-\u0bf2"
    "\u0c78-\u0c7e\u0d58-\u0d5e\u0d70-\u0d78\u0f2a-\u0f33\u1369-\u137c"
    "\u16ee-\u16f0\u17f0-\u17f9\u19da\u2070\u2074-\u2079\u2080-\u2089"
    "\u2150-\u2182\u2185-\u2189\u2460-\u249b\u24ea-\u24ff\u2776-\u2793\u2cfd"
    "\u3007\u3021-\u3029\u3038-\u303a\u3192-\u3195\u3220-\u3229\u3248-\u324f"
    "\u3251-\u325f\u3280-\u3289\u32b1-\u32bf\ua6e6-\ua6ef\ua830-\ua835"
    "\U00010107-\U00010133\U00010140-\U00010178\U0001018a\U0001018b"
    "\U000102e1-\U000102fb\U00010320-\U00010323\U00010341\U0001034a"
    "\U000103d1-\U000103d5\U00010858-\U0001085f\U00010879-\U0001087f"
    "\U000108a7-\U000108af\U000108fb-\U000108ff\U00010916-\U0001091b"
    "\U000109bc\U000109bd\U000109c0-\U000109cf\U000109d2-\U000109ff"
    "\U00010a40-\U00010a48\U00010a7d\U00010a7e\U00010a9d-\U00010a9f"
    "\U00010aeb-\U00010aef\U00010b58-\U00010b5f\U00010b78-\U00010b7f"
    "\U00010ba9-\U00010baf\U00010cfa-\U00010cff\U00010e60-\U00010e7e"
    "\U00010f1d-\U00010f26\U00010f51-\U00010f54\U00010fc5-\U00010fcb"
    "\U00011052-\U00011065\U000111e1-\U000111f4\U0001173a\U0001173b"
    "\U000118ea-\U000118f2\U00011c5a-\U00011c6c\U00011fc0-\U00011fd4"
    "\U00012400-\U0001246e\U00016b5b-\U00016b61\U00016e80-\U00016e96"
    "\U00016ff4-\U00016ff6\U0001d2c0-\U0001d2d3\U0001d2e0-\U0001d2f3"
    "\U0001d360-\U0001d378\U0001e8c7-\U0001e8cf\U0001ec71-\U0001ecab"
    "\U0001ecad-\U0001ecaf\U0001ecb1-\U0001ecb4\U0001ed01-\U0001ed2d"
    "\U0001ed2f-\U0001ed3d\U0001f100-\U0001f10c"
)
_number = r"[\d%s]" % _non_decimal_numbers  # Go's \p{N}
_letter = r"[^\W\d_%s]" % _non_decimal_numbers  # Go's \p{L}
_base_var_def = r"_*(%s+[-_]+)*%s*%s+([-_]*[^\W_]+)*" % (_number, _number, _letter)
_variable_regex = re.compile(
    r"(?<!\$)\$\{(%s(\.%s(\[[0-9]+\])*)*(\[[0-9]+\])*)\}"
    % (_base_var_def, _base_var_def)
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
