Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

The no_drift test additionally exercises removing non-required fields. Tag a field with a
`# CAN_REMOVE` comment on each of its lines; after the baseline no-drift check the test strips those
lines (`grep -v '# CAN_REMOVE'`), then asserts the resulting plan is an update and that deploying it
leaves no drift. This guards the class of bug where a field the config stops declaring is dropped from
the update request instead of being cleared. Configs without `# CAN_REMOVE` only run the baseline. If a
field's removal does not converge, tag it `# CANNOT_REMOVE` with a comment explaining the error or drift,
rather than leaving a failing `# CAN_REMOVE`. The marker is matched with its `# ` prefix, so
`# CANNOT_REMOVE` and prose that mentions the marker are neither stripped nor treated as a tag.
