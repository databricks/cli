#!/bin/bash
# uv older than 0.9.9 has no `workspace` subcommand.
if [ "$1" = "workspace" ]; then
  echo "error: unrecognized subcommand 'workspace'" >&2
  exit 2
fi
echo "fake uv: unexpected call: $*" >&2
exit 1
