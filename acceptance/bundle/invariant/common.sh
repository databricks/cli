# Shared prologue for the deploy-based invariant bodies (no_drift, redeploy, update,
# destroy_recreate). migrate reuses only the cleanup trap; canonical uses neither.

_invariant_cleanup() {
    # Destroy only what we deployed: a rejected fuzz config deployed nothing, and
    # destroying nothing hits unstubbed URLs on the local fake server.
    if [ -z "${deployed:-}" ]; then
        return
    fi

    # destroy_recreate destroys to LOG.destroy itself, so it points cleanup elsewhere.
    trace $CLI bundle destroy --auto-approve &> "${CLEANUP_LOG:-LOG.destroy}"
    cat "${CLEANUP_LOG:-LOG.destroy}" | contains.py '!panic:' '!internal error' > /dev/null

    # Run cleanup script if present. The fuzzer has no named INPUT_CONFIG, so guard
    # the lookup against the script's `set -u`.
    CLEANUP_SCRIPT="$TESTDIR/../configs/${INPUT_CONFIG:-}-cleanup.sh"
    if [ -f "$CLEANUP_SCRIPT" ]; then
        source "$CLEANUP_SCRIPT" &> LOG.cleanup
    fi
}

# Validate and deploy databricks.yml. On success sets `deployed=1` and prints
# INPUT_CONFIG_OK; a rejected config leaves `deployed` unset with the code in
# `deploy_rc`. Call on a bare line (not in if/||) so `set -e` still aborts curated tests.
invariant_deploy() {
    # We redirect output rather than record it because some configs that are being tested may produce warnings
    trace $CLI bundle validate &> LOG.validate
    cat LOG.validate | contains.py '!panic:' '!internal error' > /dev/null

    trap _invariant_cleanup EXIT

    $CLI bundle plan -o json > plan.json 2>LOG.plan_initial.err
    cat LOG.plan_initial.err | contains.py '!panic:' '!internal error' > /dev/null

    trace $CLI bundle deploy $(readplanarg plan.json) &> LOG.deploy
    deploy_rc=$?
    cat LOG.deploy | contains.py '!panic:' '!internal error' > /dev/null

    # A rejected config skips the marker below, so the fuzzer counts it as a rejection,
    # not a bug (curated tests already aborted above under `bash -e`).
    if [ "$deploy_rc" -ne 0 ]; then
        return "$deploy_rc"
    fi
    deployed=1

    # Marks a good config for the fuzzer: any failure after this is a detected bug.
    echo INPUT_CONFIG_OK
}
