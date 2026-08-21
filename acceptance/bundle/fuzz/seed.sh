# One seed: prepare chain + invariant target. Args: seed_dir seed.
cd "$1"
# Per-seed names: seeds share one workspace; leftover state looks like drift.
export UNIQUE_NAME="$UNIQUE_NAME-$2"
export FUZZ_SEED="$2"
# Same prepare chain the harness merged (root helpers like trace, then fuzz).
source "$TESTROOT/script.prepare"
source "$TESTDIR/script.prepare"
source "$INVARIANT_DIR/$FUZZ_TARGET/script"
