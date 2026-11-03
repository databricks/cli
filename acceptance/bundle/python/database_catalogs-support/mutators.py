from dataclasses import replace

from databricks.bundles.core import database_catalog_mutator
from databricks.bundles.database_catalogs import DatabaseCatalog


@database_catalog_mutator
def update_database_catalog(database_catalog: DatabaseCatalog) -> DatabaseCatalog:
    assert isinstance(database_catalog.name, str)

    return replace(database_catalog, name=f"{database_catalog.name} (updated)")
