Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

The fuzz/ test is a harness over the invariants below rather than an invariant itself: it runs
generated configs through a real invariant test script. fuzz/script only sets up the per-seed
environment; acceptance/bin/run_fuzz.py drives the seed loop and classifies each outcome as
deployed / rejected / gap / hang / bug. Both the target and the way configs are built are matrixed
in fuzz/test.toml:

`FUZZ_TARGET` picks the invariant, and each one is also a curated invariant test that runs
over the `INPUT_CONFIG` matrix:

- `no_drift` -- deploy, then no drift
- `migrate` -- Terraform deploy, migrate to direct, then no drift
- `delete_idempotent` -- deploy, delete by emptying the config, then re-run the delete on restored state
- `destroy_idempotent` -- deploy, destroy, then destroy again on restored state

`FUZZ_MODE` picks how the config is built:

- `generate` -- build a random resource by walking the live `databricks bundle schema`
- `mutate` -- perturb one of the curated configs (see MUTATE_BASES in emit_fuzz_config.py)

Free-form scalars are occasionally replaced with dangerous / near-range-end values (empty,
whitespace, over-long, control characters, int32/int64 boundaries) to probe the CLI's input
handling.

Since the schema comes from the CLI under test, an unrelated struct change can shift a
seed onto a new config. A failure is a real CLI bug (panic, internal error, or drift),
not flakiness; the failing seed's `LOG.repro` prints a ready-to-run repro, of the form
`FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_TARGET=no_drift FUZZ_MODE=generate task test-fuzz`.
