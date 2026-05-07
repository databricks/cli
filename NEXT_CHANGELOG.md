# NEXT CHANGELOG

## Release v0.300.0

### CLI

* `auth login` no longer falls back to plaintext when the OS keyring is reachable but locked. The unlock prompt shown by the probe now runs in parallel with the OAuth flow, and the token is stored in the keyring once the user has typed their password.

### Bundles

* Fixed `--force-pull` on `bundle summary` and `bundle open` so the flag bypasses the local state cache and reads state from the workspace.

### Dependency updates
