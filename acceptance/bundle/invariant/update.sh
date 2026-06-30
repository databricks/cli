# Shared invariant body: given a databricks.yml in the current directory, deploy it,
# edit one updatable field (a comment/description), and assert the redeploy issues an
# in-place update -- not a recreate -- and leaves no drift. This exercises the update
# (PATCH) path that create-only deploys never touch; a resource whose update path is
# missing or buggy shows up here as a recreate, a spurious unrelated change, or drift.
# Sourced by fuzz/script (random configs).

# The update invariant only applies to configs with an editable comment/description
# field. A random config without one isn't a bug, so skip it before deploying (no
# INPUT_CONFIG_OK marker, so the fuzzer treats it as a rejection).
if ! edit_fuzz_config.py databricks.yml --detect 2>LOG.detect.err; then
    return 0
fi

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

# A rejected config didn't deploy, so skip the INPUT_CONFIG_OK marker; otherwise the
# fuzzer reads the update/drift below as a bug. Curated tests run under `bash -e` and
# already aborted above, so this only fires in the fuzzer subshell.
if [ "$deploy_rc" -ne 0 ]; then
    return "$deploy_rc"
fi
deployed=1

# Special message to fuzzer that generated config was fine.
# Any failures after this point will be considered as "bug detected" by fuzzer.
echo INPUT_CONFIG_OK

# Change the comment/description and re-plan: this plan must show an in-place update.
edit_fuzz_config.py databricks.yml 2>LOG.edit.err
cat LOG.edit.err | contains.py '!Traceback' > /dev/null

$CLI bundle plan -o json > LOG.update_plan.json 2>LOG.update_plan.err
cat LOG.update_plan.err | contains.py '!panic' '!internal error' > /dev/null

# Apply the edit. Run it unconditionally so any panic lands in LOG.redeploy for the
# harness post-scan; whether the update is in-place and converges is gated below.
trace $CLI bundle deploy &> LOG.redeploy
redeploy_rc=$?
cat LOG.redeploy | contains.py '!panic' '!internal error' > /dev/null

# A random fuzzed config can deploy yet legitimately differ from the fake server's
# state on update, so the fuzzer sets SKIP_DRIFT_CHECK on runs where only the no-panic
# invariant is asserted.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # The fuzzer runs this with errexit off and reads the return code, so accumulate
    # failures into update_rc instead of letting the trailing no-panic check reset $?.
    update_rc=0
    [ "$redeploy_rc" -eq 0 ] || update_rc=1

    # The edit must update in place, not recreate.
    verify_plan_action.py LOG.update_plan.json update || update_rc=1

    # And the applied update must converge: a re-plan shows no further changes.
    $CLI bundle plan -o json > LOG.planjson 2>LOG.planjson.err
    cat LOG.planjson.err | contains.py '!panic' '!internal error' > /dev/null || update_rc=1
    verify_no_drift.py LOG.planjson || update_rc=1
    return "$update_rc"
fi
