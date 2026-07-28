# Shared setup for the invariant target scripts (no_drift, migrate), also reached when the
# fuzzer sources them. Renders the config (fuzz-generated when FUZZ_SEED is set, curated
# otherwise), installs the destroy-on-exit trap, and defines invariant_deploy.

if [ -n "${FUZZ_SEED:-}" ]; then
    emit_fuzz_config.py > databricks.yml 2>LOG.gen.err
    cp databricks.yml LOG.config
else
    # Copy data files to test directory
    cp -r "$TESTDIR/../data/." . &> LOG.cp

    # Run init script if present
    INIT_SCRIPT="$TESTDIR/../configs/$INPUT_CONFIG-init.sh"
    if [ -f "$INIT_SCRIPT" ]; then
        source "$INIT_SCRIPT" &> LOG.init
    fi

    envsubst < $TESTDIR/../configs/$INPUT_CONFIG > databricks.yml

    cp databricks.yml LOG.config
fi

cleanup() {
    # Destroy even when deploy failed: a deploy that died part-way still created resources.
    trace $CLI bundle destroy --auto-approve &> LOG.destroy
    cat LOG.destroy | contains.py '!panic:' '!internal error' > /dev/null

    # Run cleanup script if present
    CLEANUP_SCRIPT="$TESTDIR/../configs/${INPUT_CONFIG:-}-cleanup.sh"
    if [ -f "$CLEANUP_SCRIPT" ]; then
        source "$CLEANUP_SCRIPT" &> LOG.cleanup
    fi
}

trap cleanup EXIT

# Fuzz-only validate panic check before deploy. Curated configs skip it -- deploy runs the
# same validate pipeline. Output is redirected, not recorded, as a fuzzed config may warn.
if [ -n "${FUZZ_SEED:-}" ]; then
    trace $CLI bundle validate &> LOG.validate
    cat LOG.validate | contains.py '!panic:' '!internal error' > /dev/null
fi

# Deploy via the given command (may start with VAR=val prefixes; trace applies them).
# set -e is off only around the deploy so the panic check runs even on failure (a
# panicking-but-rejected config is a bug); a clean non-zero deploy just exits as a rejection.
# On success it prints INPUT_CONFIG_OK, after which the fuzzer treats any failure as a bug.
invariant_deploy() {
    set +e
    trace "$@" &> LOG.deploy
    deploy_rc=$?
    set -e
    cat LOG.deploy | contains.py '!panic:' '!internal error' > /dev/null
    if [ "$deploy_rc" -ne 0 ]; then
        exit "$deploy_rc"
    fi
    echo INPUT_CONFIG_OK
}
