#!/bin/bash

# Index provisioning takes 15-30 minutes, which is why this config used to be excluded from
# cloud runs entirely. None of the invariants need a queryable index -- they deploy, re-plan,
# delete and re-delete -- so cap the wait instead of skipping the coverage.
#
# Sourced by invariant_render, so the export applies to every $CLI call in the script. It has
# to live here rather than in test.toml, which is per-directory and would cap the waits of all
# the other invariant configs too.
export DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT=30
