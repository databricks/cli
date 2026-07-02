# Shared invariant body: deploy databricks.yml, destroy it, and assert a re-plan wants
# to CREATE everything again -- proving destroy cleared all tracked state. A resource
# destroy forgets shows up as "skip" (still present), a bug. Sourced by fuzz/script.

source "$TESTDIR/../common.sh"

# This body destroys to LOG.destroy itself, so the cleanup trap must log elsewhere.
CLEANUP_LOG=LOG.destroy_cleanup

invariant_deploy
if [ -z "${deployed:-}" ]; then
    return "$deploy_rc"
fi

# Destroy unconditionally so any panic lands in LOG.destroy for the post-scan;
# completeness (re-plan recreates everything) is gated below.
trace $CLI bundle destroy --auto-approve &> LOG.destroy
destroy_rc=$?
cat LOG.destroy | contains.py '!panic:' '!internal error' > /dev/null

# Clean destroy leaves nothing, so stop the trap from destroying again (unstubbed calls).
if [ "$destroy_rc" -eq 0 ]; then
    deployed=""
fi

# A fuzzed config can deploy yet legitimately leave state the re-plan reads differently,
# so the fuzzer sets SKIP_DRIFT_CHECK to assert only no-panic.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # errexit is off under the fuzzer; accumulate into recreate_rc so the trailing check can't reset $?.
    recreate_rc=0
    [ "$destroy_rc" -eq 0 ] || recreate_rc=1

    $CLI bundle plan -o json > LOG.recreate_plan.json 2>LOG.recreate_plan.err
    cat LOG.recreate_plan.err | contains.py '!panic:' '!internal error' > /dev/null || recreate_rc=1
    verify_plan_action.py LOG.recreate_plan.json create || recreate_rc=1
    return "$recreate_rc"
fi
