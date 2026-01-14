#!/usr/bin/env python3
import json
import os
import subprocess
import sys
import time

table_name = sys.argv[1]
# Extract catalog.schema from table_name
parts = table_name.split(".")
if len(parts) != 3:
    print(f"Invalid table name: {table_name}. Expected format: catalog.schema.table", file=sys.stderr)
    sys.exit(1)

catalog_name = parts[0]
schema_name = parts[1]
full_schema_name = f"{catalog_name}.{schema_name}"

cli = os.environ.get("CLI", "databricks")


def run_cli(*args):
    result = subprocess.run([cli, *args], capture_output=True, text=True)
    return result


def execute_sql(warehouse_id, sql):
    """Execute SQL using the API endpoint."""
    payload = json.dumps({"warehouse_id": warehouse_id, "statement": sql, "wait_timeout": "30s"})
    return run_cli("api", "post", "/api/2.0/sql/statements/", "--json", payload)


# Create schema if it doesn't exist
result = run_cli("schemas", "get", full_schema_name)
if result.returncode == 0:
    print(f"Schema {full_schema_name} already exists")
else:
    result = run_cli("schemas", "create", schema_name, catalog_name)
    if result.returncode != 0:
        print(f"Failed to create schema: {result.stderr}", file=sys.stderr)
        sys.exit(1)
    print(f"Created schema {full_schema_name}")

# Get warehouse ID from environment variable
warehouse_id = os.environ.get("TEST_DEFAULT_WAREHOUSE_ID")
if not warehouse_id:
    print("TEST_DEFAULT_WAREHOUSE_ID environment variable is not set", file=sys.stderr)
    sys.exit(1)

# Create a simple table
sql = f"CREATE TABLE IF NOT EXISTS {table_name} (id INT, value STRING, timestamp TIMESTAMP) USING DELTA"

result = execute_sql(warehouse_id, sql)
if result.returncode != 0:
    print(f"Failed to create table: {result.stderr}", file=sys.stderr)
    sys.exit(1)

print(f"Created table {table_name}")

# Insert some sample data so the monitor has something to analyze
insert_sql = f"""INSERT INTO {table_name} VALUES
(1, 'test1', current_timestamp()),
(2, 'test2', current_timestamp()),
(3, 'test3', current_timestamp())"""

result = execute_sql(warehouse_id, insert_sql)
if result.returncode != 0:
    print(f"Failed to insert data: {result.stderr}", file=sys.stderr)
    sys.exit(1)

print(f"Inserted sample data into {table_name}")

# Wait for table to be visible in Unity Catalog
for attempt in range(10):
    result = run_cli("tables", "get", table_name)
    if result.returncode == 0:
        table_info = json.loads(result.stdout)
        print(f"Table {table_name} is now visible (catalog_name={table_info.get('catalog_name')})")
        break
    if attempt < 9:
        time.sleep(1)
    else:
        print(f"Warning: Table may not be immediately visible: {result.stderr}", file=sys.stderr)
