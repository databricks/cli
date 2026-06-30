# Shared invariant body: given a databricks.yml in the current directory, assert that
# `bundle validate -o json` is deterministic -- two runs on the same config must
# produce byte-identical output. Catches nondeterministic map ordering or other
# unstable serialization in config loading/resolution. There is no deploy, so no
# cleanup/destroy and no cloud state. Sourced by fuzz/script (random configs).

$CLI bundle validate -o json > validate1.json 2>LOG.validate1.err
validate_rc=$?
cat LOG.validate1.err | contains.py '!panic' '!internal error' > /dev/null

# A rejected config didn't validate; that's not a bug, just an invalid fuzz config, so
# skip the INPUT_CONFIG_OK marker. Curated tests run under `bash -e` and already
# aborted above, so this only fires in the fuzzer subshell.
if [ "$validate_rc" -ne 0 ]; then
    return "$validate_rc"
fi

# Special message to fuzzer that generated config was fine.
# Any failures after this point will be considered as "bug detected" by fuzzer.
echo INPUT_CONFIG_OK

$CLI bundle validate -o json > validate2.json 2>LOG.validate2.err
cat LOG.validate2.err | contains.py '!panic' '!internal error' > /dev/null

# Determinism is cloud-independent and cheap, so unlike drift it always runs (no
# SKIP_DRIFT_CHECK gate): identical input must yield identical output regardless of the
# seed window. A diff here is a real bug, not a fake-server limitation.
diff_rc=0
diff validate1.json validate2.json > LOG.validate.diff || diff_rc=1
return "$diff_rc"
