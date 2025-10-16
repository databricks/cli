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

start_time=$(date +%s.%N)
set +e
output=$($CLI ssh connect --cluster="$CLUSTER_ID" --profile="$PROFILE" --releases-dir=./dist -- "echo 'Connection successful'" 2>&1)
exit_code=$?
set -e

if [ $exit_code -ne 0 ]; then
    echo "❌ Failed to establish ssh connection (exit code: $exit_code)"
    echo "Output: $output"
    exit 1
fi

if [[ "$output" != *"Connection successful"* ]]; then
    echo "❌ SSH connection established but output doesn't contain expected value"
    echo "Expected to contain: 'Connection successful'"
    echo "Actual: '$output'"
    exit 1
fi

end_time=$(date +%s.%N)
duration=$(echo "$end_time - $start_time" | bc)
duration_ms=$(echo "$duration * 1000" | bc)
echo "✅ Basic connectivity OK ($duration_ms ms)"

exit 0
