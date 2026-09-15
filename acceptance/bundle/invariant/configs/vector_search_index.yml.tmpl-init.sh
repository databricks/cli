#!/bin/bash

# Index provisioning takes 15-30 minutes, which is why this config used to be excluded from
# cloud runs entirely. None of the invariants need a queryable index -- they deploy, re-plan,
# delete and re-delete -- so cap the wait instead of skipping the coverage.
#
# Sourced by invariant_render, so the export applies to every $CLI call in the script.
#
# Deliberately per-config rather than Env in the directory's test.toml: this index is three
# orders of magnitude slower than the next slowest config (cluster, ~100s), and capping every
# config would leave nothing exercising a real WaitAfterCreate against a real backend.
export DATABRICKS_BUNDLE_RESOURCE_MAX_WAIT=30
