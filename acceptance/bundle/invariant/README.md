Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

The fuzz/ test is different: instead of a curated config it generates random configs
from the live `databricks bundle schema` (see fuzz/script). Because the schema is read
from the CLI under test, an unrelated change to a resource struct can shift a seed onto
a new config. A failure there is a real CLI bug (a panic, internal error, or drift), not
test flakiness; reproduce it with `FUZZ_SEED_START=<seed> FUZZ_SEED_COUNT=1 task test-fuzz`.
