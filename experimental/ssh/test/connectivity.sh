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
output=$($CLI ssh connect --shutdown-delay=10s --cluster="$CLUSTER_ID" --profile="$PROFILE" --releases-dir=./dist -- "echo 'Connection successful'" 2>&1)
exit_code=$?
set -e

echo "Output: $output"

# wait for the server to shutdown and the output to be propagated to the job run
sleep 15

# Check for job submission and extract run ID
if [[ "$output" == *"Job submitted successfully with run ID:"* ]]; then
    echo "🔍 Detected job submission, extracting run ID..."
    run_id=$(echo "$output" | grep -o "Job submitted successfully with run ID: [0-9]*" | grep -o "[0-9]*$")
    echo "Run ID: $run_id"

    echo "📊 Fetching job run details..."
    job_run_json=$($CLI jobs get-run --profile="$PROFILE" "$run_id")
    echo "$job_run_json"

    echo "🔍 Extracting task ID..."
    task_run_id=$(echo "$job_run_json" | jq -r '.tasks[0].run_id')
    echo "Task Run ID: $task_run_id"

    echo "📋 Fetching task run output..."
    $CLI jobs get-run-output --profile="$PROFILE" "$task_run_id"
fi

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
