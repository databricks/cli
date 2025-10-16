#!/bin/bash

# SSH Tunnel Connectivity Test Script
# Usage: ./experimental/ssh/test/connectivity.sh <cluster-id> [ssh-tunnel-binary-path] [profile]

set -e

CLUSTER_ID="$1"
CLI=${2:-./cli}
PROFILE="${3:-DEFAULT}"

echo "=== SSH Tunnel Test ==="
echo "Start time: $(date)"
echo "Profile: $PROFILE"
echo "Cluster ID: $CLUSTER_ID"
echo "SSH Tunnel CLI: $CLI"

echo "🔍 Testing basic connectivity..."

set +e
output=$($CLI ssh connect --cluster="$CLUSTER_ID" --profile="$PROFILE" --releases-dir=./dist -- "echo 'Connection successful'" 2>&1)
exit_code=$?
set -e

echo "Output: $output"

if [ $exit_code -ne 0 ]; then
    echo "❌ Failed to establish ssh connection (exit code: $exit_code)"
    exit 1
fi

if [[ "$output" != *"Connection successful"* ]]; then
    echo "❌ SSH connection established but output doesn't contain expected value"
    echo "Expected to contain: 'Connection successful'"
    exit 1
fi

echo "✅ Basic connectivity OK"

exit 0
