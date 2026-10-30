#!/bin/bash

# The offline materialization leaves its own tables behind, so the schema is dropped with CASCADE.
echo "Cleaning up temporary source table"
$CLI tables delete main.matset_inv_$UNIQUE_NAME.source_table || true
$CLI schemas delete main.matset_inv_$UNIQUE_NAME --force || true
