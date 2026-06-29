Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

The fuzz/ test instead generates random configs from the live `databricks bundle
schema` (see fuzz/script) and runs each one through the same no_drift.sh check the
no_drift test uses. Since the schema comes from the CLI under test, an unrelated
struct change can shift a seed onto a new config. A failure is a real CLI bug (panic,
internal error, or drift), not flakiness; reproduce with
`FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 task test-fuzz`.
