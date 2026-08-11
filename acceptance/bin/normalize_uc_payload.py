#!/usr/bin/env python3
"""
Normalize a UC JSON payload read from stdin, writing the result to stdout. The payload
is either a bundle plan or a `catalogs get` / `schemas get` response; the same pruning
applies to both (the plan-only `changes` handling simply does not fire on a GET response).

UC managed-default properties (unity.catalog.managed.<format>.defaults.*) are ALWAYS
pruned. UC auto-populates these on every catalog/schema, but only on some clouds (AWS/GCP
yes, Azure no), so a committed golden cannot show them. The CLI already classifies them as
backend defaults, so they never affect the plan; dropping them here is output-only.

Managed properties surface in three shapes, all handled by pruning matching keys:
  - a `properties` map (remote_state.properties, new_state.value.properties, ...);
    an emptied `properties` object is then dropped so {} vs absent doesn't diverge.
  - a `changes` entry keyed `properties` or `properties['unity.catalog.managed...']`;
    an emptied `changes` object is likewise dropped.

Any field names passed as arguments are additionally deleted wherever they appear. These
are the volatile server-set fields (created_at, metastore_id, schema_id, ...) each test
previously stripped with an inline `jq 'walk(del(...))'`; the set is test-specific, so it
stays at the call site rather than being centralized here.
"""

import json
import re
import sys

# Matches a managed-default property key, whether bare (as a map key) or wrapped in a
# plan change key like properties['unity.catalog.managed.delta.defaults.delta....'].
managed_re = re.compile(r"unity\.catalog\.managed\..*\.defaults\..*")


def is_managed_change(key, value):
    """Report whether a plan `changes` entry is purely a managed-default change.

    UC emits the managed-property drift either per key (change key
    `properties['unity.catalog.managed...']`) or as the whole map at the parent path
    (bare `properties` key, managed map under `remote`). Both are backend defaults the
    CLI already skips, so they are output noise.

    >>> is_managed_change("properties['unity.catalog.managed.a.defaults.b']", {"action": "skip"})
    True
    >>> is_managed_change("properties", {"action": "skip", "reason": "backend_default", "remote": {"unity.catalog.managed.a.defaults.b": "1"}})
    True
    >>> is_managed_change("properties", {"action": "update", "remote": {"custom": "v"}})
    False
    >>> is_managed_change("name", {"action": "update"})
    False
    """
    if managed_re.search(key):
        return True
    if key != "properties" or not isinstance(value, dict):
        return False
    remote = value.get("remote")
    return isinstance(remote, dict) and bool(remote) and all(managed_re.search(k) for k in remote)


def prune(node, drop_fields):
    """Recursively drop managed-default properties and the given volatile fields.

    >>> prune({"created_at": 1, "name": "c"}, {"created_at"})
    {'name': 'c'}
    >>> prune({"properties": {"unity.catalog.managed.delta.defaults.x": "true"}}, set())
    {}
    >>> prune({"properties": {"unity.catalog.managed.delta.defaults.x": "1", "k": "v"}}, set())
    {'properties': {'k': 'v'}}
    >>> prune({"changes": {"properties": {"action": "skip", "remote": {"unity.catalog.managed.a.defaults.b": "1"}}, "name": {"action": "update"}}}, set())
    {'changes': {'name': {'action': 'update'}}}
    >>> prune([{"metastore_id": "m", "full_name": "c.s"}], {"metastore_id"})
    [{'full_name': 'c.s'}]
    """
    if isinstance(node, list):
        return [prune(v, drop_fields) for v in node]
    if not isinstance(node, dict):
        return node

    result = {}
    for key, value in node.items():
        if key in drop_fields or managed_re.search(key):
            continue
        if key == "changes" and isinstance(value, dict):
            value = {k: v for k, v in value.items() if not is_managed_change(k, v)}
        pruned = prune(value, drop_fields)
        # Drop a properties/changes object emptied by managed-key removal so goldens
        # don't diverge on {} (cloud that injects) vs absent (cloud that doesn't).
        if key in ("properties", "changes") and pruned == {}:
            continue
        result[key] = pruned
    return result


def main():
    drop_fields = set(sys.argv[1:])
    data = json.load(sys.stdin)
    json.dump(prune(data, drop_fields), sys.stdout, indent=2)
    sys.stdout.write("\n")


if __name__ == "__main__":
    main()
