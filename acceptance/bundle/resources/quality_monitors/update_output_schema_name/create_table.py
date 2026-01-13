#!/usr/bin/env python3
import os
import sys
import time
from databricks.sdk import WorkspaceClient
from databricks.sdk.service.catalog import TableInfo

table_name = sys.argv[1]
# Extract catalog.schema from table_name
parts = table_name.split(".")
if len(parts) != 3:
    print(f"Invalid table name: {table_name}. Expected format: catalog.schema.table", file=sys.stderr)
    sys.exit(1)

catalog_name = parts[0]
schema_name = parts[1]
full_schema_name = f"{catalog_name}.{schema_name}"

w = WorkspaceClient()

# Create schema if it doesn't exist
try:
    w.schemas.get(full_schema_name)
    print(f"Schema {full_schema_name} already exists")
except Exception:
    w.schemas.create(name=schema_name, catalog_name=catalog_name)
    print(f"Created schema {full_schema_name}")

# Get warehouse ID from environment variable
warehouse_id = os.environ.get("TEST_DEFAULT_WAREHOUSE_ID")
if not warehouse_id:
    print("TEST_DEFAULT_WAREHOUSE_ID environment variable is not set", file=sys.stderr)
    sys.exit(1)

# Create a simple table
sql = f"CREATE TABLE IF NOT EXISTS {table_name} (id INT, value STRING, timestamp TIMESTAMP) USING DELTA"

response = w.statement_execution.execute_statement(warehouse_id=warehouse_id, statement=sql, wait_timeout="30s")

print(f"Created table {table_name}")

# Insert some sample data so the monitor has something to analyze
insert_sql = f"""
INSERT INTO {table_name} VALUES
(1, 'test1', current_timestamp()),
(2, 'test2', current_timestamp()),
(3, 'test3', current_timestamp())
"""

w.statement_execution.execute_statement(warehouse_id=warehouse_id, statement=insert_sql, wait_timeout="30s")

print(f"Inserted sample data into {table_name}")

# Wait for table to be visible in Unity Catalog
for attempt in range(10):
    try:
        table_info = w.tables.get(table_name)
        print(f"Table {table_name} is now visible (catalog_name={table_info.catalog_name})")
        break
    except Exception as e:
        if attempt < 9:
            time.sleep(1)
        else:
            print(f"Warning: Table may not be immediately visible: {e}", file=sys.stderr)
