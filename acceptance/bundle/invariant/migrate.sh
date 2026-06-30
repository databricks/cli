# Shared invariant body: given a databricks.yml in the current directory, deploy it
# with Terraform, migrate the deployment to the direct engine, and assert there is no
# drift afterwards, with no panics / internal errors along the way. Sourced by
# migrate/script (curated configs) and fuzz/script (random schema-generated configs)
# so the deploy/migrate/drift logic lives in one place.

# migrate always starts from a Terraform deployment, so drop any engine the caller
# selected (the fuzzer runs the invariant matrix with DATABRICKS_BUNDLE_ENGINE=direct).
unset DATABRICKS_BUNDLE_ENGINE

cleanup() {
    # Only destroy what we deployed. A curated config always deploys, but a random
    # fuzzed config may be rejected, and destroying nothing just makes extra API
    # calls (which fail the local fake server on unstubbed URLs).
    if [ -z "${deployed:-}" ]; then
        return
    fi

    trace $CLI bundle destroy --auto-approve &> LOG.destroy
    cat LOG.destroy | contains.py '!panic:' '!internal error' > /dev/null

    # Run cleanup script if present. The fuzzer has no named INPUT_CONFIG, so guard
    # the lookup against the script's `set -u`.
    CLEANUP_SCRIPT="$TESTDIR/../configs/${INPUT_CONFIG:-}-cleanup.sh"
    if [ -f "$CLEANUP_SCRIPT" ]; then
        source "$CLEANUP_SCRIPT" &> LOG.cleanup
    fi
}

trap cleanup EXIT

trace DATABRICKS_BUNDLE_ENGINE=terraform $CLI bundle deploy &> LOG.deploy
deploy_rc=$?
cat LOG.deploy | contains.py '!panic:' '!internal error' > /dev/null

# A rejected config didn't deploy, so skip the INPUT_CONFIG_OK marker; otherwise the
# fuzzer reads the failing migrate/drift below as a bug. Curated tests run under
# `bash -e` and already aborted above, so this only fires in the fuzzer subshell.
if [ "$deploy_rc" -ne 0 ]; then
    return "$deploy_rc"
fi
deployed=1

# Special message to fuzzer that generated config was fine.
# Any failures after this point will be considered as "bug detected" by fuzzer.
echo INPUT_CONFIG_OK

MIGRATE_ARGS=""
# The terraform provider sorts depends_on entries alphabetically by task_key on Read
# (see terraform-provider-databricks PR #3000). Since depends_on uses TypeList
# (order-sensitive), terraform plan reports positional drift when the bundle config
# specifies depends_on in a different order than the provider's sorted state.
# This is a false positive -- the logical dependencies are identical.
if [[ "${INPUT_CONFIG:-}" == "job_with_depends_on.yml.tmpl" ]]; then
    MIGRATE_ARGS="--noplancheck"
fi

trace $CLI bundle deployment migrate $MIGRATE_ARGS &> LOG.migrate

cat LOG.migrate | contains.py '!panic:' '!internal error' > /dev/null

# Drift is the whole point for the curated migrate configs, but a random fuzzed
# config can migrate yet legitimately differ from the fake server's state, so the
# fuzzer sets SKIP_DRIFT_CHECK on runs where only the no-panic invariant is asserted.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # The fuzzer runs this with errexit off and reads the return code, so accumulate
    # failures into drift_rc instead of letting the trailing no-panic check reset $?.
    drift_rc=0
    $CLI bundle plan -o json > plan.json 2>plan.json.err
    cat plan.json.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    verify_no_drift.py plan.json || drift_rc=1
    return "$drift_rc"
fi
