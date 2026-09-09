#!/usr/bin/env python3
"""Prune the cloud-dependent parts of `bundle plan -o json` so the plan can be recorded as a
golden that is identical across clouds.

Reads plan JSON on stdin, writes the pruned JSON on stdout. Per plan node it removes:

  - remote_state: the raw backend read (per-run driver/executor IPs, instance ids,
    spark_context_id, timestamps, default_tags, ...), none of which is reproducible.
  - changes entries whose reason is in --ignore-reasons (default: managed, backend_default) --
    values the backend chooses, e.g. {aws,azure,gcp}_attributes or node types, which differ by
    cloud.
  - changes entries whose field path contains any substring in --ignore-keys -- for fields a
    backend sets to different values (so the same field lands under different reasons on
    different clouds and cannot be matched by reason alone, e.g. enable_elastic_disk).

What survives is cloud-independent: the action, and the changes driven by config or policy.
"""

import argparse
import json
import sys

parser = argparse.ArgumentParser()
parser.add_argument("--ignore-reasons", default="managed,backend_default")
parser.add_argument("--ignore-keys", default="")
args = parser.parse_args()

ignore_reasons = {r for r in args.ignore_reasons.split(",") if r}
ignore_keys = [k for k in args.ignore_keys.split(",") if k]

plan = json.load(sys.stdin)

for node in plan.get("plan", {}).values():
    node.pop("remote_state", None)
    changes = node.get("changes") or {}
    for path, change in list(changes.items()):
        if change.get("reason") in ignore_reasons or any(k in path for k in ignore_keys):
            del changes[path]

json.dump(plan, sys.stdout, indent=2, sort_keys=True)
print()
