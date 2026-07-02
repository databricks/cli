# Shared invariant body: assert `bundle validate -o json` is deterministic -- two runs
# must be byte-identical. Catches unstable map ordering / serialization in config
# loading. No deploy, so no cleanup or cloud state. Sourced by fuzz/script.

$CLI bundle validate -o json > validate1.json 2>LOG.validate1.err
validate_rc=$?
cat LOG.validate1.err | contains.py '!panic:' '!internal error' > /dev/null

# A config that fails to validate is an invalid fuzz config, not a bug, so skip the
# marker (curated tests already aborted above under `bash -e`).
if [ "$validate_rc" -ne 0 ]; then
    return "$validate_rc"
fi

# Marks a good config for the fuzzer: any failure after this is a detected bug.
echo INPUT_CONFIG_OK

$CLI bundle validate -o json > validate2.json 2>LOG.validate2.err
cat LOG.validate2.err | contains.py '!panic:' '!internal error' > /dev/null

# Determinism is cloud-independent and cheap, so it always runs (no SKIP_DRIFT_CHECK
# gate): identical input must yield identical output. A diff is a real bug.
diff_rc=0
diff validate1.json validate2.json > LOG.validate.diff || diff_rc=1
return "$diff_rc"
