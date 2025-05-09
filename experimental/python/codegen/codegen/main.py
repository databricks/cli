import argparse
from dataclasses import replace
from pathlib import Path
from textwrap import dedent

import codegen.generated_dataclass as generated_dataclass
import codegen.generated_dataclass_patch as generated_dataclass_patch
import codegen.generated_enum as generated_enum
import codegen.generated_imports as generated_imports
import codegen.jsonschema as openapi
import codegen.jsonschema_patch as openapi_patch
import codegen.packages as packages

from codegen.code_builder import CodeBuilder
from codegen.generated_dataclass import GeneratedDataclass, GeneratedType
from codegen.generated_enum import GeneratedEnum


def main(output: str):
    schemas = openapi.get_schemas()
    schemas = openapi_patch.add_extra_required_fields(schemas)
    schemas = openapi_patch.remove_unsupported_fields(schemas)

    schemas = _transitively_mark_deprecated_and_private(
        packages.RESOURCE_TYPES, schemas
    )
    # first remove deprecated fields so there are more unused schemas
    schemas = _remove_deprecated_fields(schemas)
    schemas = _remove_unused_schemas(packages.RESOURCE_TYPES, schemas)

    dataclasses, enums = _generate_code(schemas)

    generated_dataclass_patch.reorder_required_fields(dataclasses)
    generated_dataclass_patch.quote_recursive_references(dataclasses)

    _write_code(dataclasses, enums, output)

    for resource in packages.RESOURCE_TYPES:
        reachable = _collect_reachable_schemas([resource], schemas)

        resource_dataclasses = {k: v for k, v in dataclasses.items() if k in reachable}
        resource_enums = {k: v for k, v in enums.items() if k in reachable}

        _write_exports(resource, resource_dataclasses, resource_enums, output)


def _transitively_mark_deprecated_and_private(
    roots: list[str],
    schemas: dict[str, openapi.Schema],
) -> dict[str, openapi.Schema]:
    """
    If schema is only used through deprecated (private) fields, make it as deprecated (private).

    For example, if a field is marked as private, and is excluded from documentation, corresponding
    dataclasses and enums should be private as well.
    """

    not_private = _collect_reachable_schemas(roots, schemas, include_private=False)
    not_deprecated = _collect_reachable_schemas(
        roots, schemas, include_deprecated=False
    )
    new_schemas = {}

    for schema_name, schema in schemas.items():
        if schema_name not in not_private:
            schema.stage = openapi.Stage.PRIVATE

        if schema_name not in not_deprecated:
            schema.deprecated = True

        new_schemas[schema_name] = schema

    return new_schemas


def _remove_deprecated_fields(
    schemas: dict[str, openapi.Schema],
) -> dict[str, openapi.Schema]:
    new_schemas = {}

    for name, schema in schemas.items():
        if schema.type == openapi.SchemaType.OBJECT:
            new_properties = {}
            for field_name, field in schema.properties.items():
                if field.deprecated:
                    continue

                new_properties[field_name] = field

            new_schemas[name] = replace(schema, properties=new_properties)
        else:
            new_schemas[name] = schema

    return new_schemas


def _generate_code(
    schemas: dict[str, openapi.Schema],
) -> tuple[dict[str, GeneratedDataclass], dict[str, GeneratedEnum]]:
    dataclasses = {}
    enums = {}

    for schema_name, schema in schemas.items():
        if schema.type == openapi.SchemaType.OBJECT:
            generated = generated_dataclass.generate_dataclass(schema_name, schema)

            dataclasses[schema_name] = generated
        elif schema.type == openapi.SchemaType.STRING:
            generated = generated_enum.generate_enum(schema_name, schema)

            enums[schema_name] = generated
        else:
            raise ValueError(f"Unknown type: {schema.type}")

    return dataclasses, enums


def _write_exports(
    root: str,
    dataclasses: dict[str, GeneratedDataclass],
    enums: dict[str, GeneratedEnum],
    output: str,
):
    exports = []

    for _, dataclass in dataclasses.items():
        exports += [
            dataclass.class_name,
            f"{dataclass.class_name}Dict",
            f"{dataclass.class_name}Param",
        ]

    for _, enum in enums.items():
        exports += [enum.class_name, f"{enum.class_name}Param"]

    exports.sort()

    b = CodeBuilder()

    b.append("__all__ = [\n")
    for export in exports:
        b.indent().append_repr(export).append(",").newline()
    b.append("]").newline()
    b.newline()
    b.newline()

    generated_imports.append_dataclass_imports(b, dataclasses, exclude_packages=[])
    generated_imports.append_enum_imports(b, enums, exclude_packages=[])

    # FIXME should be better generalized
    if root == "resources.Job":
        _append_resolve_recursive_imports(b)

    root_package = packages.get_package(root)
    assert root_package

    # transform databricks.bundles.jobs._models.job -> databricks/bundles/jobs
    package_path = Path(root_package.replace(".", "/")).parent.parent

    source_path = Path(output) / package_path / "__init__.py"
    source_path.parent.mkdir(exist_ok=True, parents=True)
    source_path.write_text(b.build())

    print(f"Writing exports into {source_path}")


def _append_resolve_recursive_imports(b: CodeBuilder):
    """
    Resolve forward references for recursive imports so we can assume that there are no forward references
    while inspecting type annotations.
    """

    b.append(
        dedent("""
            def _resolve_recursive_imports():
                import typing

                from databricks.bundles.core._variable import VariableOr
                from databricks.bundles.jobs._models.task import Task

                ForEachTask.__annotations__ = typing.get_type_hints(
                    ForEachTask,
                    globalns={"Task": Task, "VariableOr": VariableOr},
                )

            _resolve_recursive_imports()
        """)
    )


def _collect_typechecking_imports(
    generated: GeneratedDataclass,
) -> dict[str, list[str]]:
    out = {}

    def visit_type(type_name: GeneratedType):
        if type_name.name.startswith('"'):
            out[type_name.package] = out.get(type_name.package, [])
            out[type_name.package].append(type_name.name.strip('"'))

        for parameter in type_name.parameters:
            visit_type(parameter)

    for field in generated.fields:
        visit_type(field.type_name)
        visit_type(field.param_type_name)

    return out


def _collect_reachable_schemas(
    roots: list[str],
    schemas: dict[str, openapi.Schema],
    include_private: bool = True,
    include_deprecated: bool = True,
) -> set[str]:
    """
    Remove schemas that are not reachable from the roots, because we
    don't want to generate code for them.
    """

    reachable = set(packages.PRIMITIVES)
    stack = []

    for root in roots:
        stack.append(root)

    while stack:
        current = stack.pop()
        if current in reachable:
            continue

        reachable.add(current)

        schema = schemas[current]

        if schema.type == openapi.SchemaType.OBJECT:
            for field in schema.properties.values():
                if field.ref:
                    name = field.ref.split("/")[-1]

                    if not include_private and field.stage == openapi.Stage.PRIVATE:
                        continue

                    if not include_deprecated and field.deprecated:
                        continue

                    if name not in reachable:
                        stack.append(name)

    return reachable


def _remove_unused_schemas(
    roots: list[str],
    schemas: dict[str, openapi.Schema],
) -> dict[str, openapi.Schema]:
    """
    Remove schemas that are not reachable from the roots, because we
    don't want to generate code for them.
    """

    reachable = _collect_reachable_schemas(roots, schemas)

    return {k: v for k, v in schemas.items() if k in reachable}


def _write_code(
    dataclasses: dict[str, GeneratedDataclass],
    enums: dict[str, GeneratedEnum],
    output: str,
):
    package_code = {}
    typechecking_imports = {}

    for schema_name, generated in dataclasses.items():
        package = generated.package
        code = generated_dataclass.get_code(generated)

        typechecking_imports[package] = _collect_typechecking_imports(generated)
        typechecking_imports[package]["typing_extensions"] = ["Self"]

        package_code[package] = package_code.get(package, "")
        package_code[package] += "\n" + code

    for schema_name, generated in enums.items():
        package = generated.package
        code = generated_enum.get_code(generated)

        package_code[package] = package_code.get(package, "")
        package_code[package] += "\n" + code

    package_code = {
        package: generated_imports.get_code(
            dataclasses,
            enums,
            # don't import package from itself
            exclude_packages=[package],
            typechecking_imports=typechecking_imports.get(package, {}),
        )
        + code
        for package, code in package_code.items()
    }

    for package, code in package_code.items():
        package_path = package.replace(".", "/")
        source_path = Path(output) / (package_path + ".py")

        source_path.parent.mkdir(exist_ok=True, parents=True)
        source_path.write_text(code)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument("--output", type=str)
    args = parser.parse_args()

    main(args.output)
