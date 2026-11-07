from dataclasses import replace

from codegen.generated_dataclass import (
    GeneratedDataclass,
    GeneratedField,
    GeneratedType,
)


def reorder_required_fields(models: dict[str, GeneratedDataclass]):
    """
    Reorder fields in dataclasses so that required fields come first.
    It's necessary for kwargs in the constructor to work correctly.
    """
    for name, model in models.items():
        if not model.fields:
            continue

        required_fields = [field for field in model.fields if _is_required(field)]
        optional_fields = [field for field in model.fields if not _is_required(field)]

        models[name] = replace(model, fields=required_fields + optional_fields)


def quote_recursive_references(models: dict[str, GeneratedDataclass], root: str):
    """
    If there is a cycle between two dataclasses, we need to quote one of them.

    Example:

        class Foo:
            bar: Optional[Bar]

        class Bar:
            foo: "Foo"

    Cycles are found with a DFS from the resource root, and the reference that
    points back to a dataclass still being visited is quoted, e.g. ForEachTask.task
    in jobs (Job -> Task -> ForEachTask -> Task).
    """

    # see also _append_resolve_recursive_imports

    names_by_package = {model.package: name for name, model in models.items()}
    visiting = set()
    visited = set()
    back_references: dict[str, set[str]] = {}

    def visit(name: str):
        visiting.add(name)

        for field in models[name].fields:
            for package in _referenced_packages(field.type_name):
                # None for enums and core types, they can't form a cycle
                ref = names_by_package.get(package)

                if ref in visiting:
                    model = models[ref]
                    back_references.setdefault(name, set()).update(
                        {model.class_name, f"{model.class_name}Param"}
                    )
                elif ref and ref not in visited:
                    visit(ref)

        visiting.remove(name)
        visited.add(name)

    visit(root)

    for name, references in back_references.items():
        models[name] = _quote_recursive_references_for_model(models[name], references)


def _referenced_packages(type_name: GeneratedType) -> list[str]:
    packages = [type_name.package] if type_name.package else []

    for param in type_name.parameters:
        packages += _referenced_packages(param)

    return packages


def _quote_recursive_references_for_model(
    model: GeneratedDataclass,
    references: set[str],
) -> GeneratedDataclass:
    def update_type_name(type_name: GeneratedType):
        if type_name.name in references:
            return replace(
                type_name,
                name=f'"{type_name.name}"',
            )
        elif type_name.parameters:
            return replace(
                type_name,
                parameters=[update_type_name(param) for param in type_name.parameters],
            )
        else:
            return type_name

    def update_field(field: GeneratedField):
        return replace(
            field,
            type_name=update_type_name(field.type_name),
            param_type_name=update_type_name(field.param_type_name),
        )

    return replace(
        model,
        fields=[update_field(field) for field in model.fields],
    )


def _is_required(field: GeneratedField) -> bool:
    return field.default is None and field.default_factory is None
