# NEXT CHANGELOG

## Release v0.300.0

### CLI

* `databricks auth profiles` now distinguishes "validation failed" from "couldn't validate". The JSON output adds a `status` field (`valid`, `invalid`, `unknown`, or `unvalidated`) and an `error` description for non-valid profiles. The legacy `valid` field is still emitted as `true` when validation succeeded and `false` when the profile is provably bad (auth/config error); it is omitted for transient/unknown cases that previously misreported as `valid: false`. Each profile is validated with a 10s timeout so a single dead host no longer stalls the listing.

### Bundles

### Dependency updates
