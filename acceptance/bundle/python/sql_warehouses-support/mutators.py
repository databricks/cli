from dataclasses import replace

from databricks.bundles.core import sql_warehouse_mutator
from databricks.bundles.sql_warehouses import SqlWarehouse


@sql_warehouse_mutator
def update_sql_warehouse(sql_warehouse: SqlWarehouse) -> SqlWarehouse:
    assert isinstance(sql_warehouse.name, str)

    return replace(sql_warehouse, name=f"{sql_warehouse.name} (updated)")
