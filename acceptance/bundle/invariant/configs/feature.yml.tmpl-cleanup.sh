#!/bin/bash

echo "Cleaning up temporary source table"
$CLI tables delete main.feat_inv_$UNIQUE_NAME.source_table || true
$CLI schemas delete main.feat_inv_$UNIQUE_NAME || true
