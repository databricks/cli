#!/bin/bash

echo "Cleaning up temporary schema"
$CLI schemas delete main.fspec_inv_$UNIQUE_NAME --force || true
