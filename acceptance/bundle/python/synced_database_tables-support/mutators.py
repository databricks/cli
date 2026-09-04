from dataclasses import replace

from databricks.bundles.core import synced_database_table_mutator
from databricks.bundles.synced_database_tables import SyncedDatabaseTable


@synced_database_table_mutator
def update_synced_database_table(synced_database_table: SyncedDatabaseTable) -> SyncedDatabaseTable:
    assert isinstance(synced_database_table.name, str)

    return replace(synced_database_table, name=f"{synced_database_table.name} (updated)")
