#!/bin/bash

# Clean up the temporary source table
echo "Cleaning up temporary source table"
$CLI tables delete main.test_synced_$UNIQUE_NAME.trips_source || true
$CLI schemas delete main.test_synced_$UNIQUE_NAME || true
