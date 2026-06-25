import inspect
import re
import typing
from dataclasses import is_dataclass, fields
from inspect import Signature
from types import NoneType, UnionType
from typing import get_origin, Union, get_args

import sphinx.util.inspect as sphinx_inspect
from sphinx.addnodes import pending_xref

from sphinx.application import Sphinx
from sphinx.util.inspect import stringify_signature
from sphinx.util.typing import ExtensionMetadata

from databricks.bundles.core._resource_type import _ResourceType
from databricks.bundles.core._transform import _unwrap_variable, _unwrap_optional


def get_arg_name(arg):
    if hasattr(arg, "__forward_arg__"):
        return arg.__forward_arg__

    return arg.__name__


def simplify_union_type(args: tuple, unwrap_variable):
    names = [get_arg_name(arg) for arg in args]

    if len(args) == 2 and NoneType not in args:
        if names[0] == names[1] + "Dict":
            return args[1]

    if len(args) == 2 and NoneType not in args:
        if names[0] == "Literal":
            return args[1]

    if len(args) == 3 and args[2] == NoneType:
        if names[0] == names[1] + "Dict":
            return args[1] | None

    is_optional = NoneType in args

    for arg in args:
        if nested := _unwrap_optional(arg):
            if is_optional:
                return simplify_type(nested, unwrap_variable=unwrap_variable) | None
            else:
                return simplify_type(nested, unwrap_variable=unwrap_variable)

        if unwrap_variable:
            if nested := _unwrap_variable(arg):
                if is_optional:
                    return simplify_type(nested, unwrap_variable=unwrap_variable) | None
                else:
                    return simplify_type(nested, unwrap_variable=unwrap_variable)

    return Union[args]


def simplify_type(type, unwrap_variable=True):
    """
    Simplifies type signatures.

    Rules:
    - simplify_type(list[T])    -> list[simplify_type(T)]
    - simplify_type(dict[T, U]) -> dict[simplify_type(T), simplify_type(U)]
    - simplify_type(A | B)      -> simplify_type(A) | simplify_type(B)
    - simplify_type(VariableOrOptional[T]) -> simplify_type(T) | None
    - simplify_type(Variable[T])           -> simplify_type(T)
    - simplify_type(XxxParam)              -> Xxx
    """

    origin = get_origin(type)

    if origin == list:
        arg = simplify_type(get_args(type)[0], unwrap_variable=unwrap_variable)
        arg = _unwrap_optional(arg) or arg

        return list[arg]
    elif origin == dict:
        arg0 = simplify_type(get_args(type)[0], unwrap_variable=unwrap_variable)
        arg1 = simplify_type(get_args(type)[1], unwrap_variable=unwrap_variable)

        arg0 = _unwrap_optional(arg0) or arg0
        arg1 = _unwrap_optional(arg1) or arg1

        return dict[arg0, arg1]
    elif origin in [Union, UnionType]:
        return simplify_union_type(get_args(type), unwrap_variable=unwrap_variable)
    else:
        return type


def is_inherited(cls, field_name):
    for base in inspect.getmro(cls)[1:]:
        if hasattr(base, "__annotations__") and field_name in base.__annotations__:
            return True

    return False


def stringify_annotation(annotation, mode: str = "fully-qualified-except-typing"):
    import sphinx.util.typing

    return sphinx.util.typing.stringify_annotation(annotation, mode)


def resolve_forward_ref(obj, sig):
    # resolve forward references, because some types are recursive

    import databricks.bundles.core
    import databricks.bundles.jobs._models.task
    import typing_extensions

    hints = typing.get_type_hints(
        obj,
        localns={
            "Self": typing_extensions.Self,
            "TaskParam": databricks.bundles.jobs._models.task.TaskParam,
            "VariableOr": databricks.bundles.core.VariableOr,
        },
    )

    return sig.replace(
        parameters=[
            param.replace(annotation=hints[name])
            for name, param in sig.parameters.items()
        ]
    )


def process_signature(app, what, name, obj, options, signature, return_annotation):
    if what in ("class"):
        annotations = app.env.temp_data.setdefault("annotations", {})
        annotation = annotations.setdefault(name, {})

        if is_dataclass(obj):
            for field in fields(obj):
                field_type = obj.__annotations__.get(field.name, field.type)
                field_type = simplify_type(field_type)

                if is_inherited(obj, field.name):
                    if field.name in annotation:
                        del annotation[field.name]

                    if field.name in obj.__annotations__:
                        del obj.__annotations__[field.name]
                else:
                    annotation[field.name] = stringify_annotation(
                        field_type, mode="smart"
                    )
                    obj.__annotations__[field.name] = field_type

        return "", ""

    if what in ("decorator", "method", "function"):
        sig = sphinx_inspect.signature(obj)

        # do not simplify variables in core signatures because they matter
        if name.startswith("databricks.bundles.core."):
            # return signature, return_annotation
            sig = simplify_sig(sig, unwrap_variable=False)
        elif name.startswith("databricks.bundles.") and name.endswith(".as_dict"):
            sig = simplify_as_dict_sig(sig)
        elif name.startswith("databricks.bundles.") and name.endswith(".from_dict"):
            sig = simplify_from_dict_sig(sig)
        # this is the only recursive type we have, resolution is
        elif name == "databricks.bundles.jobs.ForEachTask.create":
            sig = resolve_forward_ref(obj, sig)
            sig = simplify_sig(sig, unwrap_variable=True)
        else:
            sig = simplify_sig(sig, unwrap_variable=True)

        signature = stringify_signature(sig, show_return_annotation=False)
        return_annotation = stringify_annotation(sig.return_annotation, mode="smart")

        return signature, return_annotation


def simplify_as_dict_sig(sig: Signature):
    """
    Simplifies the signature of `as_dict` methods.

    We don't output type dict types because they are not documented separately,
    as they exactly match dataclass fields.

    Input:

        class Foo:
            def as_dict(self) -> FooDict: ...

    Output:

        class Foo:
            def as_dict(self) -> dict
    """

    return sig.replace(return_annotation=dict)


def simplify_from_dict_sig(sig: Signature):
    """
    Simplifies from_dict signature similar to as_dict.

    Input:
        class Foo:

            def from_dict(self) -> FooDict: ...

    Output:
        class Foo:

            def from_dict(self) -> dict: ...

    """

    return sig.replace(
        parameters=[param.replace(annotation=dict) for param in sig.parameters.values()]
    )


def process_docstring(app, what, name, obj, options, lines):
    for i in range(len(lines)):
        line = lines[i]

        lines[i] = re.sub(
            # turn markdown links into reST links
            r"\[([^\]]+)\]\((https?://[^\)]+)\)",
            r"`\1 <\2>`_",
            line,
        )


def simplify_sig(sig, unwrap_variable: bool) -> inspect.Signature:
    parameters = [
        param.replace(
            annotation=simplify_type(param.annotation, unwrap_variable=unwrap_variable)
        )
        if param.annotation is not param.empty
        else param
        for name, param in sig.parameters.items()
    ]

    parameters = [
        param.replace(default=inspect.Parameter.empty)
        if param.default is None
        else param
        for param in parameters
    ]

    parameters = [param for param in parameters if param.name != "self"]

    return sig.replace(parameters=parameters)


rewrite_aliases = {
    "databricks.bundles.core._bundle._T": "databricks.bundles.core.T",
    "databricks.bundles.core._variable._T": "databricks.bundles.core.T",
    "databricks.bundles.core._diagnostics._T": "databricks.bundles.core.T",
    "databricks.bundles.core._resource_mutator._T": "databricks.bundles.core.T",
}

# use dataclasses instead of typed dicts used in databricks.bundles.core
for tpe in _ResourceType.all():
    rewrite_aliases[tpe.resource_type.__name__ + "Param"] = (
        tpe.resource_type.__module__ + "." + tpe.resource_type.__name__
    )


def resolve_internal_aliases(app, doctree):
    """
    Applies rewrites for type aliases that are not correctly
    handled by Sphinx and cause broken links.
    """

    pending_xrefs = doctree.traverse(condition=pending_xref)

    for node in pending_xrefs:
        alias = node.get("reftarget", None)

        if rewrite := rewrite_aliases.get(alias):
            node["reftarget"] = rewrite


def disable_sphinx_overloads():
    # See  https://github.com/sphinx-doc/sphinx/issues/10351
    from sphinx.pycode import parser

    def add_overload_entry(self, func):
        pass

    parser.VariableCommentPicker.add_overload_entry = add_overload_entry


def skip_member(app, what, name, obj, skip, options):
    # skip databricks.bundles.<resource>._module.FooDict classes
    # because we already document Foo dataclass that is equivalent.
    if what == "module" and name.endswith("Dict") and "._models." in obj.__module__:
        return True

    return skip


def setup(app: Sphinx) -> ExtensionMetadata:
    import databricks.bundles.jobs
    import databricks.bundles.core

    # disable support for overloads, because Sphinx doesn't handle them well
    disable_sphinx_overloads()

    # instead, select the first overload manually
    for tpe in _ResourceType.all():
        mutator_fn = getattr(databricks.bundles.core, tpe.singular_name + "_mutator")
        overloads = typing.get_overloads(mutator_fn)

        setattr(databricks.bundles.core, tpe.singular_name + "_mutator", overloads[0])

    app.setup_extension("sphinx.ext.autodoc")

    app.connect("autodoc-process-signature", process_signature)
    app.connect("autodoc-process-docstring", process_docstring)
    app.connect("autodoc-skip-member", skip_member)
    app.connect("doctree-read", resolve_internal_aliases)

    return {
        "version": "1",
        "parallel_read_safe": True,
    }
