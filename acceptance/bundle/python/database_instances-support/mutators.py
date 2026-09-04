from dataclasses import replace

from databricks.bundles.database_instances import DatabaseInstance
from databricks.bundles.core import database_instance_mutator


@database_instance_mutator
def update_database_instance(database_instance: DatabaseInstance) -> DatabaseInstance:
    assert isinstance(database_instance.name, str)

    return replace(database_instance, name=f"{database_instance.name} (updated)")
