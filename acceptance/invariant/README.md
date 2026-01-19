Invariant tests are acceptance tests that can be run against many configs to check for certains properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no\_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.
