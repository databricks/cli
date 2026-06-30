# Shared invariant body: given a databricks.yml in the current directory, deploy it
# and assert there is no drift afterwards, with no panics / internal errors along
# the way. Sourced by no_drift/script (curated configs) and fuzz/script (random
# schema-generated configs) so the deploy/drift/destroy logic lives in one place.

# We redirect output rather than record it because some configs that are being tested may produce warnings
trace $CLI bundle validate &> LOG.validate

cat LOG.validate | contains.py '!panic' '!internal error' > /dev/null

cleanup() {
    # Only destroy what we deployed. A curated config always deploys, but a random
    # fuzzed config may be rejected, and destroying nothing just makes extra API
    # calls (which fail the local fake server on unstubbed URLs).
    if [ -z "${deployed:-}" ]; then
        return
    fi

    trace $CLI bundle destroy --auto-approve &> LOG.destroy
    cat LOG.destroy | contains.py '!panic' '!internal error' > /dev/null

    # Run cleanup script if present. The fuzzer has no named INPUT_CONFIG, so guard
    # the lookup against the script's `set -u`.
    CLEANUP_SCRIPT="$TESTDIR/../configs/${INPUT_CONFIG:-}-cleanup.sh"
    if [ -f "$CLEANUP_SCRIPT" ]; then
        source "$CLEANUP_SCRIPT" &> LOG.cleanup
    fi
}

trap cleanup EXIT

$CLI bundle plan -o json > plan.json 2>LOG.plan_initial.err
cat LOG.plan_initial.err | contains.py '!panic' '!internal error' > /dev/null

trace $CLI bundle deploy $(readplanarg plan.json) &> LOG.deploy
deploy_rc=$?
cat LOG.deploy | contains.py '!panic' '!internal error' > /dev/null

# A rejected config didn't deploy, so skip the INPUT_CONFIG_OK marker; otherwise
# the fuzzer reads the re-plan's "needs create" as drift. Curated tests run under
# `bash -e` and already aborted above, so this only fires in the fuzzer subshell.
if [ "$deploy_rc" -ne 0 ]; then
    return "$deploy_rc"
fi
deployed=1

# Special message to fuzzer that generated config was fine.
# Any failures after this point will be considered as "bug detected" by fuzzer.
echo INPUT_CONFIG_OK

# Drift is the whole point for the curated no_drift configs, but a random fuzzed
# config can deploy yet legitimately differ from the fake server's state, so the
# fuzzer sets SKIP_DRIFT_CHECK on runs where only the no-panic invariant is asserted.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # Check both text and JSON plan for no changes (may be >1 unchanged resource).
    # The fuzzer runs this with errexit off and reads the return code, so accumulate
    # failures into drift_rc instead of letting the trailing no-panic check reset $?.
    drift_rc=0
    $CLI bundle plan -o json > LOG.planjson 2>LOG.planjson.err
    cat LOG.planjson.err | contains.py '!panic' '!internal error' > /dev/null || drift_rc=1
    verify_no_drift.py LOG.planjson || drift_rc=1

    $CLI bundle plan 2>LOG.plan.err | contains.py '!panic' '!internal error' 'Plan: 0 to add, 0 to change, 0 to delete' > LOG.plan || drift_rc=1
    cat LOG.plan.err | contains.py '!panic' '!internal error' > /dev/null || drift_rc=1
    return "$drift_rc"
fi
