# One seed: prepare chain + invariant target. Args: seed_dir seed.
cd "$1"
# Per-seed names: seeds share one workspace, so leftover state otherwise looks like drift.
export UNIQUE_NAME="$UNIQUE_NAME-$2"
export FUZZ_SEED="$2"
# seed.sh does not walk the harness prepare chain: root helpers, parent invariant, then fuzz overrides.
source "$TESTROOT/script.prepare"
source "$TESTDIR/../script.prepare"
source "$TESTDIR/script.prepare"
source "$INVARIANT_DIR/$FUZZ_TARGET/script"
