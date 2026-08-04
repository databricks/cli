Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

Each target below runs over the `INPUT_CONFIG` matrix in test.toml:

- `no_drift` -- deploy, then no drift
- `migrate` -- Terraform deploy, migrate to direct, then no drift
- `delete_idempotent` -- deploy, delete by emptying the config, then re-run the delete on restored state
- `destroy_idempotent` -- deploy, destroy, then destroy again on restored state

In order to add a new test, add a config to configs/ and include it in test.toml.

The helpers in script.prepare are also sourced from outside this directory: ../fuzz runs generated
configs through these same target scripts. It sets `INVARIANT_DIR` (which defaults to this
directory) so configs/ and data/ resolve from there, and redefines `invariant_render` to generate a
config instead of rendering one from configs/. Keep that in mind when changing the helpers.
