Deleting a `grants` or `permissions` block, or emptying it to `[]`, now revokes what it
granted. Both were previously ignored: the grant or permission stayed in place, and so did
anything added outside the bundle. An empty list and a removed block are the same request,
and both are now applied as a delete ([#6474](https://github.com/databricks/cli/pull/6474)).

`grants: []` revokes everything, and so does `permissions: []` - except on jobs, pipelines and
SQL warehouses, where the API requires exactly one owner on every update, so the owner is kept.
That owner is read from the object rather than assumed to be whoever deploys, so an object
owned by a service principal, or handed over since it was deployed, keeps the owner it has.

Unaffected: a bundle that has always had an empty list, deleting the resource a block belongs
to, and `secret_scopes` permissions, which already revoked down to the deploying user.
