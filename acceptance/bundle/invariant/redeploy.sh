# Shared invariant body: deploy databricks.yml, then deploy again and assert the second
# deploy is a clean no-op. Catches create handlers that don't round-trip their inputs
# (or mutators that re-derive a field), which surface as a redeploy wanting to change or
# recreate. Sourced by fuzz/script (random configs).

source "$TESTDIR/../common.sh"

invariant_deploy
if [ -z "${deployed:-}" ]; then
    return "$deploy_rc"
fi

# Deploy again, unconditionally, so any panic lands in LOG.redeploy for the post-scan;
# convergence (success + no drift) is gated below.
trace $CLI bundle deploy &> LOG.redeploy
redeploy_rc=$?
cat LOG.redeploy | contains.py '!panic:' '!internal error' > /dev/null

# A fuzzed config can deploy yet legitimately fail to redeploy or differ, so the fuzzer
# sets SKIP_DRIFT_CHECK to assert only no-panic.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # errexit is off under the fuzzer; accumulate into drift_rc so the trailing check can't reset $?.
    drift_rc=0
    [ "$redeploy_rc" -eq 0 ] || drift_rc=1

    # Check both text and JSON plan for no changes (may be >1 unchanged resource).
    $CLI bundle plan -o json > LOG.planjson 2>LOG.planjson.err
    cat LOG.planjson.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    verify_no_drift.py LOG.planjson || drift_rc=1

    $CLI bundle plan 2>LOG.plan.err | contains.py '!panic:' '!internal error' 'Plan: 0 to add, 0 to change, 0 to delete' > LOG.plan || drift_rc=1
    cat LOG.plan.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    return "$drift_rc"
fi
