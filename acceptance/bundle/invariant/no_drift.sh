# Shared invariant body: deploy databricks.yml and assert no drift, no panics. Sourced
# by no_drift/script (curated configs) and fuzz/script (random configs).

source "$TESTDIR/../common.sh"

invariant_deploy
if [ -z "${deployed:-}" ]; then
    return "$deploy_rc"
fi

# A fuzzed config can deploy yet legitimately differ from the fake server, so the
# fuzzer sets SKIP_DRIFT_CHECK to assert only no-panic; curated configs check drift.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # Check both text and JSON plan for no changes. errexit is off under the fuzzer, so
    # accumulate into drift_rc; the trailing no-panic check must not reset $?.
    drift_rc=0
    $CLI bundle plan -o json > LOG.planjson 2>LOG.planjson.err
    cat LOG.planjson.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    verify_no_drift.py LOG.planjson || drift_rc=1

    $CLI bundle plan 2>LOG.plan.err | contains.py '!panic:' '!internal error' 'Plan: 0 to add, 0 to change, 0 to delete' > LOG.plan || drift_rc=1
    cat LOG.plan.err | contains.py '!panic:' '!internal error' > /dev/null || drift_rc=1
    return "$drift_rc"
fi
