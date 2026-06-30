# Shared invariant body: given a databricks.yml in the current directory, deploy it,
# destroy it, and assert a re-plan wants to CREATE every resource again -- proving the
# destroy cleared all tracked state with nothing orphaned. A resource that destroy
# forgets to remove from state shows up here as a "skip" (still considered present),
# which is a bug. Sourced by fuzz/script (random configs).

# We redirect output rather than record it because some configs that are being tested may produce warnings
trace $CLI bundle validate &> LOG.validate

cat LOG.validate | contains.py '!panic' '!internal error' > /dev/null

cleanup() {
    # Only destroy what we deployed. The body destroys on the happy path and clears
    # `deployed`, so this trap only fires when deploy or destroy failed partway.
    if [ -z "${deployed:-}" ]; then
        return
    fi

    trace $CLI bundle destroy --auto-approve &> LOG.destroy_cleanup
    cat LOG.destroy_cleanup | contains.py '!panic' '!internal error' > /dev/null

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

# A rejected config didn't deploy, so skip the INPUT_CONFIG_OK marker; otherwise the
# fuzzer reads the destroy/recreate below as a bug. Curated tests run under `bash -e`
# and already aborted above, so this only fires in the fuzzer subshell.
if [ "$deploy_rc" -ne 0 ]; then
    return "$deploy_rc"
fi
deployed=1

# Special message to fuzzer that generated config was fine.
# Any failures after this point will be considered as "bug detected" by fuzzer.
echo INPUT_CONFIG_OK

# Destroy unconditionally so any panic lands in LOG.destroy for the harness post-scan;
# whether the destroy was complete (re-plan recreates everything) is gated below.
trace $CLI bundle destroy --auto-approve &> LOG.destroy
destroy_rc=$?
cat LOG.destroy | contains.py '!panic' '!internal error' > /dev/null

# On a clean destroy nothing remains, so stop the trap from destroying again (which
# would just make unstubbed API calls against the fake server).
if [ "$destroy_rc" -eq 0 ]; then
    deployed=""
fi

# A random fuzzed config can deploy yet legitimately leave fake-server state that the
# re-plan reads differently, so the fuzzer sets SKIP_DRIFT_CHECK on runs where only the
# no-panic invariant is asserted.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # The fuzzer runs this with errexit off and reads the return code, so accumulate
    # failures into recreate_rc instead of letting the trailing no-panic check reset $?.
    recreate_rc=0
    [ "$destroy_rc" -eq 0 ] || recreate_rc=1

    $CLI bundle plan -o json > LOG.recreate_plan.json 2>LOG.recreate_plan.err
    cat LOG.recreate_plan.err | contains.py '!panic' '!internal error' > /dev/null || recreate_rc=1
    verify_plan_action.py LOG.recreate_plan.json create || recreate_rc=1
    return "$recreate_rc"
fi
