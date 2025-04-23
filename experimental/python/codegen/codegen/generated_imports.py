from textwrap import dedent

import codegen.packages as packages
from codegen.code_builder import CodeBuilder
from codegen.generated_dataclass import GeneratedDataclass
from codegen.generated_enum import GeneratedEnum


def append_enum_imports(
    b: CodeBuilder,
    enums: dict[str, GeneratedEnum],
    exclude_packages: list[str],
) -> None:
    for schema_name in enums.keys():
        package = packages.get_package(schema_name)
        class_name = packages.get_class_name(schema_name)

        if package in exclude_packages:
            continue

        b.append(f"from {package} import {class_name}, {class_name}Param\n").newline()


def append_dataclass_imports(
    b: CodeBuilder,
    dataclasses: dict[str, GeneratedDataclass],
    exclude_packages: list[str],
) -> None:
    for schema_name in dataclasses.keys():
        package = packages.get_package(schema_name)
        class_name = packages.get_class_name(schema_name)

        if package in exclude_packages:
            continue

        b.append(
            f"from {package} import {class_name}, {class_name}Dict, {class_name}Param"
        ).newline()


def get_code(
    dataclasses: dict[str, GeneratedDataclass],
    enums: dict[str, GeneratedEnum],
    typechecking_imports: dict[str, list[str]],
    exclude_packages: list[str],
) -> str:
    b = CodeBuilder()

    b.append(
        "from typing import Literal, Optional, TypedDict, ClassVar, TYPE_CHECKING\n"
    )
    b.append("from enum import Enum\n")
    b.append("from dataclasses import dataclass, replace, field\n")
    b.append("\n")
    b.append("from databricks.bundles.core._resource import Resource\n")
    b.append("from databricks.bundles.core._transform import _transform\n")
    b.append(
        "from databricks.bundles.core._transform_to_json import _transform_to_json_value\n"
    )
    b.append(
        "from databricks.bundles.core._variable import VariableOr, VariableOrOptional, VariableOrList, VariableOrDict\n"
    )
    b.newline()

    runtime_dataclasses = {
        k: v
        for k, v in dataclasses.items()
        if v.class_name not in typechecking_imports.get(v.package, [])
    }

    append_dataclass_imports(b, runtime_dataclasses, exclude_packages)
    append_enum_imports(b, enums, exclude_packages)

    # typechecking_imports is special case because it's only for TYPE_CHECKING
    # and formatter doesn't eliminate unused imports for TYPE_CHECKING
    if typechecking_imports:
        b.newline()
        b.append("if TYPE_CHECKING:").newline()
        for package, imports in typechecking_imports.items():
            b.indent().append(f"from {package} import {', '.join(imports)}").newline()

    return b.build()
