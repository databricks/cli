This is a harness over the invariant tests in ../invariant rather than an invariant itself: it runs
generated configs through a real invariant target script. script only sets up the per-seed
environment; acceptance/bin/run_fuzz.py drives the seed loop and classifies each outcome as
deployed / rejected / gap / hang / bug. Both the target and the way configs are built are matrixed
in test.toml:

`FUZZ_TARGET` picks the invariant, and each one is also a curated invariant test that runs over the
`INPUT_CONFIG` matrix:

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

The invariant helpers come from ../invariant/script.prepare, which script.prepare sources directly
because test.toml and script.prepare only merge along the directory chain. For the same reason the
server stubs and ignore patterns this test needs are copied into test.toml.

Since the schema comes from the CLI under test, an unrelated struct change can shift a
seed onto a new config. A failure is a real CLI bug (panic, internal error, or drift),
not flakiness; the failing seed's `LOG.repro` prints a ready-to-run repro, of the form
`FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_TARGET=no_drift FUZZ_MODE=generate task test-fuzz`.
