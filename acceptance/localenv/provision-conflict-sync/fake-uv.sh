#!/bin/bash
#
# Fake uv for the provision-conflict-sync acceptance test. It satisfies the
# version probe and the Python install, then fails `uv sync` with a trimmed
# sample of uv's real resolver-conflict stderr so the CLI reaches the new
# E_PROVISION_CONFLICT classification without a real (network-dependent) sync.
# The test script renames this to "uv" and puts it first on PATH, mirroring
# acceptance/cmd/psql.
#
case "$1" in
--version)
	echo "uv 0.0.0-fake"
	;;
python)
	# `uv python install <minor>`: pretend the interpreter is available.
	;;
sync)
	echo "error: No solution found when resolving dependencies:" >&2
	echo "  Because the project's requirements are unsatisfiable, we cannot proceed." >&2
	exit 1
	;;
esac
exit 0
