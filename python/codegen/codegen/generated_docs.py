#!/usr/bin/env python3
"""Generate the Sphinx .rst pages from the resource module dirs under
databricks/bundles/, so the documented resource list can never drift from the
generated code. Driven by scanning the output tree, not RESOURCE_NAMESPACE.

The doc structure lives in the *.rst.tmpl templates so it can be reviewed
separately from this code: doc_page.rst.tmpl (one page per resource) and
doc_index.rst.tmpl (the index prose header + generated toctree)."""

from pathlib import Path
from string import Template

import codegen.packages as packages

# .title() mangles acronyms; override those namespaces.
_TITLE_OVERRIDES = {
    "sql_warehouses": "SQL Warehouses",
    "mcp_services": "MCP Services",
}


def _load_template(name: str) -> Template:
    return Template((Path(__file__).parent / name).read_text())


_PAGE_TEMPLATE = _load_template("doc_page.rst.tmpl")
_INDEX_TEMPLATE = _load_template("doc_index.rst.tmpl")


def _title(namespace: str) -> str:
    return _TITLE_OVERRIDES.get(namespace, namespace.replace("_", " ").title())


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
        module = packages.get_root_package(namespace)
        page = _PAGE_TEMPLATE.substitute(title=_title(namespace), module=module)
        (docs / f"databricks.bundles.{namespace}.rst").write_text(page)

    entries = ["databricks.bundles.core"] + [
        packages.get_root_package(ns) for ns in namespaces
    ]
    toctree = "\n".join(f"   {entry}" for entry in entries)
    (docs / "index.rst").write_text(_INDEX_TEMPLATE.substitute(toctree=toctree))

    print(f"Writing {len(namespaces) + 1} doc pages into {docs}")
