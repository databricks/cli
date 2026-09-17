#!/usr/bin/env python3
"""Generate the Sphinx .rst pages from the resource module dirs under
databricks/bundles/, so the documented resource list can never drift from the
generated code. Driven by scanning the output tree, not RESOURCE_NAMESPACE."""

from pathlib import Path

import codegen.packages as packages

# .title() mangles acronyms; override those namespaces.
_TITLE_OVERRIDES = {
    "sql_warehouses": "SQL Warehouses",
    "mcp_services": "MCP Services",
}

# Fixed-width underlines matching the hand-written pages so the resource pages
# that already exist regenerate byte-identical.
_H1 = "=" * 31
_H2 = "-" * 15

# Prose header kept verbatim from the hand-written index.rst; only the toctree
# below it is generated.
_INDEX_HEADER = """\
databricks-bundles
--------------------------------

`databricks-bundles` package implements Python support for Declarative Automation Bundles.

See `What is Python support for Declarative Automation Bundles? (TBD) <#>`_.


.. toctree::
   :maxdepth: 7
"""


def _title(namespace: str) -> str:
    return _TITLE_OVERRIDES.get(namespace, namespace.replace("_", " ").title())


def _page(namespace: str) -> str:
    module = packages.get_root_package(namespace)
    return (
        f"{_title(namespace)}\n"
        f"{_H1}\n"
        "\n"
        f".. currentmodule:: {module}\n"
        "\n"
        f"**Package:** ``{module}``\n"
        "\n"
        "Classes\n"
        f"{_H2}\n"
        "\n"
        f".. automodule:: {module}\n"
    )


def write_docs(output: str):
    docs = Path(output) / "docs"
    bundles = Path(output) / "databricks" / "bundles"

    # core is hand-written; every other package dir gets a generated page.
    namespaces = sorted(
        p.name
        for p in bundles.iterdir()
        if p.name != "core" and (p / "__init__.py").exists()
    )

    # Drop stale pages so a removed resource loses its page; keep core.rst.
    for rst in docs.glob("databricks.bundles.*.rst"):
        if rst.name != "databricks.bundles.core.rst":
            rst.unlink()

    for namespace in namespaces:
        (docs / f"databricks.bundles.{namespace}.rst").write_text(_page(namespace))

    entries = ["databricks.bundles.core"] + [
        packages.get_root_package(ns) for ns in namespaces
    ]
    toctree = "".join(f"   {entry}\n" for entry in entries)
    (docs / "index.rst").write_text(_INDEX_HEADER + "\n" + toctree)

    print(f"Writing {len(namespaces) + 1} doc pages into {docs}")
