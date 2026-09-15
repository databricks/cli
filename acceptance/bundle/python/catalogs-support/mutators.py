from dataclasses import replace

from databricks.bundles.catalogs import Catalog
from databricks.bundles.core import catalog_mutator


@catalog_mutator
def update_catalog(catalog: Catalog) -> Catalog:
    assert isinstance(catalog.comment, str)

    return replace(catalog, comment=f"{catalog.comment} (updated)")
