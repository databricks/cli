Deleting a `grants` or `permissions` block, or emptying it to `[]`, now revokes what it
granted. Both were previously ignored: the grant or permission stayed in place, and so did
anything added outside the bundle. An empty list and a removed block are the same request,
and both are now applied as a delete ([#6474](https://github.com/databricks/cli/pull/6474)).

`grants: []` revokes everything. `permissions: []` revokes everything but the object owner,
which the API requires on every update. The owner is read from the object rather than assumed
to be whoever deploys, so an object owned by a service principal, or handed over since it was
deployed, keeps the owner it has.

Nothing changes for a bundle that has always had an empty list, and deleting the resource a
block belongs to is unaffected.
