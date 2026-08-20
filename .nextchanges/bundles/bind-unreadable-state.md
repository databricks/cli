Fix `bundle deployment bind` silently binding over a deployment state it could not read, which could take over a resource that was already managed by the bundle.
