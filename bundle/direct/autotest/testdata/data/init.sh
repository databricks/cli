#!/bin/bash
# A cluster init script that does nothing, uploaded with the bundle so a fixture declaring one names a
# file that exists. A real workspace fails the cluster otherwise -- "Init scripts failed ... Tree node
# with path ... does not exist" -- where the mock server accepts any path.
echo "init"
