# Shared invariant body: deploy databricks.yml, edit a comment/description, and assert
# the redeploy is an in-place update, not a recreate, with no drift. Exercises the
# update (PATCH) path create-only deploys never touch. Sourced by fuzz/script.

source "$TESTDIR/../common.sh"

# Only configs with an editable comment/description apply here; skip others before
# deploying (no marker, so the fuzzer treats it as a rejection, not a bug).
if ! edit_fuzz_config.py databricks.yml --detect 2>LOG.detect.err; then
    return 0
fi

invariant_deploy
if [ -z "${deployed:-}" ]; then
    return "$deploy_rc"
fi

# Change the comment/description and re-plan: this plan must show an in-place update.
edit_fuzz_config.py databricks.yml 2>LOG.edit.err
cat LOG.edit.err | contains.py '!Traceback' > /dev/null

$CLI bundle plan -o json > LOG.update_plan.json 2>LOG.update_plan.err
cat LOG.update_plan.err | contains.py '!panic:' '!internal error' > /dev/null

# Apply the edit, unconditionally, so any panic lands in LOG.redeploy for the post-scan;
# in-place update and convergence are gated below.
trace $CLI bundle deploy &> LOG.redeploy
redeploy_rc=$?
cat LOG.redeploy | contains.py '!panic:' '!internal error' > /dev/null

# A fuzzed config can deploy yet legitimately differ on update, so the fuzzer sets
# SKIP_DRIFT_CHECK to assert only no-panic.
if [ -z "${SKIP_DRIFT_CHECK:-}" ]; then
    # errexit is off under the fuzzer; accumulate into update_rc so the trailing check can't reset $?.
    update_rc=0
    [ "$redeploy_rc" -eq 0 ] || update_rc=1

    # The edit must update in place, not recreate.
    verify_plan_action.py LOG.update_plan.json update || update_rc=1

    # And the applied update must converge: a re-plan shows no further changes.
    $CLI bundle plan -o json > LOG.planjson 2>LOG.planjson.err
    cat LOG.planjson.err | contains.py '!panic:' '!internal error' > /dev/null || update_rc=1
    verify_no_drift.py LOG.planjson || update_rc=1
    return "$update_rc"
fi
