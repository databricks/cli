#!/bin/bash
#
# This script prints its arguments and exits.
# The test script renames this script to "psql" in order to capture the arguments that the CLI passes to psql command.
#
echo "echo-arguments.sh was called with the following arguments: $@"
echo "PGPASSWORD=${PGPASSWORD}"
echo "PGSSLMODE=${PGSSLMODE}"
exit 0
