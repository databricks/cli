Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/. Name it after the key under `resources:` of the type
it is about: `<key>.yml.tmpl` (for example `jobs.yml.tmpl`), or `<key>_<variant>.yml.tmpl` for a variant.
`TestInvariantConfigNames` checks the names, and `TestInvariantConfigsCoverage` requires a `<key>.yml.tmpl`
for every resource type. Every `configs/*.yml.tmpl` runs in every invariant test
(`EnvMatrixFiles` in test.toml). If a config cannot run in some tests, add `configs/<config>-exclude.toml`
with an `[[Exclude]]` entry per reason: `Tests` names the tests that skip it (all if omitted), `When` the
further conditions, such as `CONFIG_Cloud=true` or `READPLAN=1`, and `Reason` says why. See
`EnvMatrixFiles` in `acceptance/internal/config.go`.

The no_drift test additionally exercises removing non-required fields. Tag a field with a
`# CAN_REMOVE` comment on each of its lines; after the baseline no-drift check the test strips those
lines (`grep -v '# CAN_REMOVE'`), then asserts the resulting plan is an update and that deploying it
leaves no drift. This guards the class of bug where a field the config stops declaring is dropped from
the update request instead of being cleared. Configs without `# CAN_REMOVE` only run the baseline. If a
field's removal does not converge, tag it `# CANNOT_REMOVE` with a comment explaining the error or drift,
rather than leaving a failing `# CAN_REMOVE`. The marker is matched with its `# ` prefix, so
`# CANNOT_REMOVE` and prose that mentions the marker are neither stripped nor treated as a tag.
