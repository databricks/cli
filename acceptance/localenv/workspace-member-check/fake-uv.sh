#!/bin/bash
# The workspace root is the test folder, where this script is installed as uv.
if [ "$1" = "workspace" ] && [ "$2" = "dir" ]; then
  cd "$(dirname "$0")" && pwd
  exit 0
fi
echo "fake uv: unexpected call: $*" >&2
exit 1
