Invariant tests are acceptance tests that can be run against many configs to check for certain properties.
Unlike regular acceptance tests full output is not recorded, unless the condition is not met. For example,
no_drift test checks that there are no actions planned after successful deploy. If that's not the case, the
test will dump full JSON plan to the output.

In order to add a new test, add a config to configs/ and include it in test.toml.

## Field mutations

`apply_update` and `apply_remote_update` check that a single field change is planned,
applied, and converges. They read the change to make from the config itself, as a
trailing comment, so the same config still works for every other target:

```yaml
      comment: base comment            # ACTION: SET "changed comment"
      #description: test description   # ACTION: UNCOMMENT
      user_api_scopes: [iam.access]    # ACTION: REMOVE
```

`apply_update` edits the config, so the plan must show a *local* change for that
field. `apply_remote_update` deploys the edit and then restores the pre-deploy state
snapshot, leaving the remote ahead of both state and config -- so the plan must show
a *remote* change for that field. Checking which of the two it is matters: reverting
a config also produces a change, so the action alone proves nothing.

Both targets return the remote to the base config afterwards, which for REMOVE and
UNCOMMENT means clearing the field. A resource whose update request drops empty
fields never converges, so only SET is usable there (see `schema.yml.tmpl`).

A field the engine deliberately ignores expects `skip` instead, and which target
sees that depends on why it is ignored -- `ignore_local_changes` is skipped when the
config changes, `ignore_remote_changes` when the remote drifts:

```yaml
      #compute_size: LARGE             # ACTION: UNCOMMENT EXPECT_REMOTE_UPDATE: skip
```

Run one mutation on its own with `$MUTATION`:

```
MUTATION=comment:set go test ./acceptance \
  -run 'TestAccept/bundle/invariant/apply_update/.*INPUT_CONFIG=schema.yml.tmpl'
```

See `acceptance/bin/apply_mutation.py` for the full annotation reference.
