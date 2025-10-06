#!/bin/bash
#
# This script prints its arguments and exits.
# The test script renames this script to "psql" in order to capture the arguments that the CLI passes to psql command.
#
echo "Simulating connection failure with exit code '2'"
exit 2
