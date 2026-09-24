#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 BINARY EXPECTED_VERSION" >&2
  exit 2
fi

binary=$1
expected=$2
actual=$($binary version --output json | jq -r .version)
if [[ "$actual" != "$expected" ]]; then
  echo "binary version mismatch: expected $expected, got $actual" >&2
  exit 1
fi
