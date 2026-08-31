#!/usr/bin/env python3
"""Snapshot the typed public API surface of databricks.bundles.core, for regression-guarding.

Usage (from the acceptance script in this directory, inside the databricks-bundles uv env):

    uv run --python 3.11 $UV_ARGS python dump_public_api.py

Why only core: the resource model namespaces (jobs, pipelines, ...) are entirely
pydabs-codegen output, already guarded byte-for-byte by CI's `generate-check`. The core
wiring — Resources, the *_mutator functions, the _ResourceType registry, __all__ — is
HAND-WRITTEN and is exactly what the codegen/wiring refactor converts to generated code.
This dumps that surface to a golden `output.txt` so the refactor (and future ones) can't
silently drop a type hint, move a `*` marker, rename a method, or change the export set.

Determinism notes:
  * Types are rendered by their PUBLIC SHORT NAME (`Variable[str]`, `Location`, `None`)
    rather than `inspect`/`repr`'s fully-qualified internal module path
    (`databricks.bundles.core._variable.Variable`). This is deliberate: the refactor moves
    internal modules around, and a golden keyed on internal paths would fail even when the
    public API is unchanged. Short names change only when the public type actually changes.
  * Signatures are reconstructed from `inspect.Signature` (not its string form) so
    positional-only `/`, keyword-only `*`, `*args` and `**kwargs` markers render explicitly.
  * Module symbols, methods and properties are sorted; declaration order is preserved only
    where it is part of the contract (dataclass fields, enum members).
  * The uv invocation pins `--python 3.11` so the golden generated locally with `-update`
    matches CI, since resolved type reprs can differ across versions.
"""

import collections.abc
import dataclasses
import enum
import inspect
import sys
import types
import typing


def render_type(t) -> str:
    """Render a type annotation by public short name, module-location independent."""
    if t is None or t is type(None):
        return "None"
    if t is Ellipsis:
        return "..."
    if isinstance(t, str):
        # A forward-ref written as a string literal in the source (e.g. "JobParam").
        return t
    if isinstance(t, typing.ForwardRef):
        return t.__forward_arg__
    if isinstance(t, typing.TypeVar):
        return t.__name__

    origin = typing.get_origin(t)
    args = typing.get_args(t)

    if origin is not None:
        if origin is typing.Union or origin is types.UnionType:
            return "Union[" + ", ".join(render_type(a) for a in args) + "]"
        if origin is typing.Literal:
            return "Literal[" + ", ".join(repr(a) for a in args) + "]"
        if origin is collections.abc.Callable:
            if not args:
                return "Callable"
            # get_args(Callable[[int], str]) == ([int], str); [0] is the arg list.
            params, ret = args[0], args[-1]
            params_str = "..." if params is Ellipsis else "[" + ", ".join(render_type(a) for a in params) + "]"
            return "Callable[" + params_str + ", " + render_type(ret) + "]"
        name = _short_name(origin)
        if args:
            return name + "[" + ", ".join(render_type(a) for a in args) + "]"
        return name

    return _short_name(t)


def _short_name(t) -> str:
    return getattr(t, "__name__", None) or getattr(t, "_name", None) or str(t)


def render_signature(func) -> str:
    """Reconstruct a signature string with explicit / * ** markers and short types."""
    sig = inspect.signature(func)
    parts = []
    last_kind = None
    emitted_star = False
    for p in sig.parameters.values():
        if last_kind == inspect.Parameter.POSITIONAL_ONLY and p.kind != inspect.Parameter.POSITIONAL_ONLY:
            parts.append("/")
        if p.kind == inspect.Parameter.KEYWORD_ONLY and not emitted_star:
            parts.append("*")
            emitted_star = True

        s = p.name
        if p.kind == inspect.Parameter.VAR_POSITIONAL:
            s = "*" + s
            emitted_star = True
        elif p.kind == inspect.Parameter.VAR_KEYWORD:
            s = "**" + s

        if p.annotation is not inspect.Parameter.empty:
            s += ": " + render_type(p.annotation)
        if p.default is not inspect.Parameter.empty:
            sep = " = " if p.annotation is not inspect.Parameter.empty else "="
            s += sep + repr(p.default)
        parts.append(s)
        last_kind = p.kind

    if last_kind == inspect.Parameter.POSITIONAL_ONLY:
        parts.append("/")

    ret = ""
    if sig.return_annotation is not inspect.Signature.empty:
        ret = " -> " + render_type(sig.return_annotation)
    return "(" + ", ".join(parts) + ")" + ret


def _bases(cls) -> str:
    # Skip object and private (underscore) bases, mirroring _members(): a generated private
    # base like _GeneratedResources is an implementation detail, not the public contract.
    names = [b.__name__ for b in cls.__bases__ if b is not object and not b.__name__.startswith("_")]
    return "(" + ", ".join(names) + ")" if names else ""


def _members(cls, predicate):
    return sorted((name, obj) for name, obj in inspect.getmembers(cls, predicate) if not name.startswith("_"))


def render_class(name, cls, out):
    if isinstance(cls, type) and issubclass(cls, enum.Enum):
        out.append(f"class {name}(Enum):")
        for member in cls:
            out.append(f"    {member.name} = {member.value!r}")
        out.append("")
        return

    out.append(f"class {name}{_bases(cls)}:")
    if dataclasses.is_dataclass(cls):
        for f in dataclasses.fields(cls):
            line = f"    {f.name}: {render_type(f.type)}"
            if f.default is not dataclasses.MISSING:
                line += f" = {f.default!r}"
            elif f.default_factory is not dataclasses.MISSING:
                line += " = <factory>"
            out.append(line)

    # classmethods (e.g. create_error) surface as bound methods, not plain functions.
    for m_name, m in _members(cls, inspect.ismethod):
        out.append(f"    @classmethod def {m_name}{render_signature(m)}")
    for m_name, m in _members(cls, inspect.isfunction):
        out.append(f"    def {m_name}{render_signature(m)}")
    for p_name, prop in _members(cls, lambda x: isinstance(x, property)):
        ret = ""
        if prop.fget is not None:
            r = inspect.signature(prop.fget).return_annotation
            if r is not inspect.Signature.empty:
                ret = " -> " + render_type(r)
        out.append(f"    @property {p_name}{ret}")
    out.append("")


def render_symbol(name, obj, out):
    if inspect.isclass(obj):
        render_class(name, obj, out)
    elif inspect.isfunction(obj):
        overloads = typing.get_overloads(obj)
        for ov in overloads:
            out.append(f"@overload def {name}{render_signature(ov)}")
        out.append(f"def {name}{render_signature(obj)}")
        out.append("")
    else:
        # Type aliases (VariableOr*), rendered by structure.
        out.append(f"{name} = {render_type(obj)}")
        out.append("")


def render_registry(out):
    # _ResourceType is intentionally not exported from core, but the registry it builds is
    # part of the wiring the refactor regenerates, so snapshot it too.
    from databricks.bundles.core._resource_type import _ResourceType

    out.append("== _ResourceType.all() registry ==")
    for rt in sorted(_ResourceType.all(), key=lambda rt: rt.singular_name):
        out.append(
            f"singular_name={rt.singular_name} plural_name={rt.plural_name} resource_type={rt.resource_type.__name__}"
        )
    out.append("")


def main():
    import databricks.bundles.core as core

    out = ["== module databricks.bundles.core =="]
    out.append("__all__ = [")
    for name in sorted(core.__all__):
        out.append(f"    {name},")
    out.append("]")
    out.append("")

    for name in sorted(core.__all__):
        render_symbol(name, getattr(core, name), out)

    render_registry(out)

    sys.stdout.write("\n".join(out) + "\n")


if __name__ == "__main__":
    main()
