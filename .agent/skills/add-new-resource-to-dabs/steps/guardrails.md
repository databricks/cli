# Guardrails

If the resource is non-ephemeral in nature, deletion should trigger a warning + confirmation. Implement this in [deploy.go](./bundle/phases/deploy.go).

Likewise, if deleting + recreating the resource causes significant downtime (e.g. vector search index requiring re-indexing post-deploy), this should trigger a warning + confirmation, too.
