This is a harness over the invariant tests in ../invariant rather than an invariant itself: it runs
mutated configs through a real invariant target script. script only sets up the per-seed
environment; acceptance/bin/run_fuzz.py drives the seed loop and classifies each outcome as
deployed / rejected / gap / hang / bug. The target is matrixed in test.toml:

`FUZZ_TARGET` picks the invariant, and each one is also a curated invariant test that runs over the
`INPUT_CONFIG` matrix:

- `no_drift` -- deploy, then no drift
- `migrate` -- Terraform deploy, migrate to direct, then no drift
- `delete_idempotent` -- deploy, delete by emptying the config, then re-run the delete on restored state
- `destroy_idempotent` -- deploy, destroy, then destroy again on restored state

Each seed perturbs one of the curated configs in MUTATE_BASES (see emit_fuzz_config.py): delete or
replace existing fields, and (via the live `databricks bundle schema`) inject valid optional fields
the base omits. Free-form scalars are occasionally replaced with dangerous / near-range-end values
(empty, whitespace, over-long, control characters, int32/int64 boundaries) to probe the CLI's input
handling.

The invariant helpers come from ../invariant/script.prepare, which script.prepare sources directly
because test.toml and script.prepare only merge along the directory chain. For the same reason the
server stubs and ignore patterns this test needs are copied into test.toml; script asserts the two
stub sets stay in sync.

A mutated config can reach an API route the testserver does not model, which is a coverage gap
rather than a missing stub. test.toml answers those with a per-method catch-all stub returning a
`TESTSERVER_GAP` marker, so the seed is recorded as a gap instead of failing the run, and the seed's
log names the route the CLI could not reach.

Since the schema comes from the CLI under test, an unrelated struct change can shift a
seed onto a new config. A failure is a real CLI bug (panic, internal error, or drift),
not flakiness; the failing seed's `LOG.repro` prints a ready-to-run repro, of the form
`ENVFILTER=FUZZ_TARGET=no_drift FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_CHECK_DRIFT=0 task test-fuzz`.
The target goes through `ENVFILTER` because it is a matrix key: set as a plain env var the harness
overrides it and re-runs all four variants.

`FUZZ_CHECK_DRIFT` is part of the repro because it selects the oracle: at `0` (the committed run)
`invariant_verify_no_drift` is replaced with a plan-determinism diff, and at `1` (`task test-fuzz`
and the nightly) the exact check from ../invariant runs unchanged. Only the committed run is
expected to be green: the wide drift-on window stops at the first open finding, so a red scheduled
run is a bug to triage rather than a regression in the change that happened to trigger it.
