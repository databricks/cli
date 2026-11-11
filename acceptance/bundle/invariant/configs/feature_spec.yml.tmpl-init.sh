#!/bin/bash

# Creating a feature spec resolves its lookup table in Unity Catalog, and the invariant harness
# provisions no tables of its own.
echo "Creating temporary schema and feature table: main.fspec_inv_$UNIQUE_NAME"

$CLI schemas create fspec_inv_$UNIQUE_NAME main -o json | jq '{full_name}'
create_feature_spec_fixtures.py main.fspec_inv_$UNIQUE_NAME
