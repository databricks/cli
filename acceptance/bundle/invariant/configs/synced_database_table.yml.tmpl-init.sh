#!/bin/bash

# Create a unique source table for this test run to avoid hitting the 20-table-per-source limit
echo "Creating temporary source table: main.test_synced_$UNIQUE_NAME.trips_source"

# Create schema using CLI
$CLI schemas create test_synced_$UNIQUE_NAME main -o json | jq '{full_name}'

# Create source table from samples.nyctaxi.trips using SQL API
# MSYS_NO_PATHCONV=1 prevents Git Bash on Windows from converting /api/... to C:/Program Files/Git/api/...
MSYS_NO_PATHCONV=1 $CLI api post "/api/2.0/sql/statements/" --json "{
    \"warehouse_id\": \"$TEST_DEFAULT_WAREHOUSE_ID\",
    \"statement\": \"CREATE TABLE main.test_synced_$UNIQUE_NAME.trips_source AS SELECT * FROM samples.nyctaxi.trips LIMIT 10\",
    \"wait_timeout\": \"45s\"
  }" > /dev/null
