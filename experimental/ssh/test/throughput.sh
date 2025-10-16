#!/bin/bash

# SSH Tunnel Performance Test Script
# Usage:
# 1. Setup ssh config: ./cli ssh setup --cluster --name
# 2. Run: ./experimental/ssh/test/throughput.sh <cluster-id> <ssh-config-hostname> [ssh-tunnel-binary-path] [profile]

set -e

CLUSTER_ID="$1"
HOSTNAME="$2"
CLI=${3:-./cli}
PROFILE="${4:-DEFAULT}"

TEST_SIZES=(300 600)  # MB

if [ -z "$CLUSTER_ID" ] || [ -z "$HOSTNAME" ]; then
    echo "Usage: $0 <cluster-id> <hostname> [ssh-tunnel-binary-path] [profile]"
    exit 1
fi

echo "=== SSH Tunnel Test ==="
echo "Cluster ID: $CLUSTER_ID"
echo "Hostname: $HOSTNAME"
echo "Profile: $PROFILE"
echo "Start time: $(date)"
echo "SSH Tunnel: $CLI"
echo "⚡ Testing Throughput..."

# Create test files
echo "Creating test files..."
for size in "${TEST_SIZES[@]}"; do
    if [ ! -f "test_${size}mb.dat" ]; then
        dd if=/dev/zero of="test_${size}mb.dat" bs=1M count=$size 2>/dev/null
        echo "  Created test_${size}mb.dat"
    fi
done
echo

# Upload tests
echo "📤 Upload Speed Tests:"
for size in "${TEST_SIZES[@]}"; do
    echo -n "  ${size}MB file: "
    scp "test_${size}mb.dat" "$HOSTNAME:/tmp/test_upload_${size}mb.dat"
done
echo

# Download tests
echo "📥 Download Speed Tests:"
for size in "${TEST_SIZES[@]}"; do
    echo -n "  ${size}MB file: "
    scp "$HOSTNAME:/tmp/test_upload_${size}mb.dat" "./test_download_${size}mb.dat"
done
echo

# Cleanup
echo "🧹 Cleaning up..."
for size in "${TEST_SIZES[@]}"; do
    rm -f "test_${size}mb.dat" "test_download_${size}mb.dat"
    $CLI ssh connect --cluster="$CLUSTER_ID" --profile="$PROFILE" -- "rm -f /tmp/test_upload_${size}mb.dat" 2>/dev/null || true
done
echo
