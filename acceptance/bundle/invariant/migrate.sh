# Shared invariant body: deploy databricks.yml with Terraform, migrate to the direct
# engine, and assert no drift, no panics. Sourced by migrate/script (curated configs)
# and fuzz/script (random configs).

# migrate always starts from a Terraform deployment, so drop any engine the caller
# selected (the fuzzer runs the invariant matrix with DATABRICKS_BUNDLE_ENGINE=direct).
unset DATABRICKS_BUNDLE_ENGINE

source "$TESTDIR/../common.sh"

trap _invariant_cleanup EXIT

trace DATABRICKS_BUNDLE_ENGINE=terraform $CLI bundle deploy &> LOG.deploy
deploy_rc=$?
cat LOG.deploy | contains.py '!panic:' '!internal error' > /dev/null

# A rejected config skips the marker below, so the fuzzer counts it as a rejection, not
# a bug (curated tests already aborted above under `bash -e`).
if [ "$deploy_rc" -ne 0 ]; then
    return "$deploy_rc"
fi
deployed=1

# Marks a good config for the fuzzer: any failure after this is a detected bug.
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

# A fuzzed config can migrate yet legitimately differ from the fake server, so the
# fuzzer sets SKIP_DRIFT_CHECK to assert only no-panic; curated configs check drift.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # errexit is off under the fuzzer; accumulate into drift_rc so the trailing check can't reset $?.
    drift_rc=0
    $CLI bundle plan -o json > plan.json 2>plan.json.err
    cat plan.json.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    verify_no_drift.py plan.json || drift_rc=1
    return "$drift_rc"
fi
