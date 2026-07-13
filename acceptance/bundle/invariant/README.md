Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

The fuzz/ test generates random configs from the live `databricks bundle schema`
(see fuzz/script) and runs each one through a real invariant test script. The target is
selected by `FUZZ_TARGET` (matrixed in fuzz/test.toml); each target is also a curated
invariant test that runs over the `INPUT_CONFIG` matrix. `FUZZ_RESOURCE_COUNT` (also
matrixed in fuzz/test.toml) controls how many resources each generated config contains;
with more than one, the generator links them with `${resources.*}` references (each
resource referencing an earlier one, so the graph stays acyclic) so the interpolation and
deploy-ordering paths are exercised. Free-form scalars are occasionally replaced with
dangerous / near-range-end values (empty, whitespace, over-long, control characters,
int32/int64 boundaries) to probe the CLI's input handling.

- `no_drift` -- deploy, then no drift
- `migrate` -- Terraform deploy, migrate to direct, then no drift
- `redeploy` -- deploy twice; the second deploy must be a no-op
- `canonical` -- `validate -o json` must be byte-identical across two runs
- `update` -- edit a comment/description; the redeploy must update in place (not recreate)
- `destroy_recreate` -- deploy then destroy; a re-plan must recreate everything

Since the schema comes from the CLI under test, an unrelated struct change can shift a
seed onto a new config. A failure is a real CLI bug (panic, internal error, or drift),
not flakiness; reproduce with
`FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 FUZZ_TARGET=no_drift task test-fuzz`.
For a multi-resource repro, add `FUZZ_RESOURCE_COUNT=2`.
