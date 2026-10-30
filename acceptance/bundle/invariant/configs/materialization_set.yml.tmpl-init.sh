#!/bin/bash

# The feature's delta_table_source must already exist, and the invariant harness provisions no
# tables of its own.
echo "Creating temporary source table: main.matset_inv_$UNIQUE_NAME.source_table"

$CLI schemas create matset_inv_$UNIQUE_NAME main -o json | jq '{full_name}'

# MSYS_NO_PATHCONV=1 prevents Git Bash on Windows from converting /api/... to a filesystem path.
MSYS_NO_PATHCONV=1 $CLI api post "/api/2.0/sql/statements/" --json "{
    \"warehouse_id\": \"$TEST_DEFAULT_WAREHOUSE_ID\",
    \"statement\": \"CREATE TABLE main.matset_inv_$UNIQUE_NAME.source_table (id INT, value STRING, timestamp TIMESTAMP) USING DELTA\",
    \"wait_timeout\": \"45s\"
  }" > /dev/null
