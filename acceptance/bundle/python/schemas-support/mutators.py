from dataclasses import replace

from databricks.bundles.core import schema_mutator
from databricks.bundles.schemas import Schema


@schema_mutator
def update_schema(schema: Schema) -> Schema:
    assert isinstance(schema.name, str)

    return replace(schema, name=f"{schema.name} (updated)")
