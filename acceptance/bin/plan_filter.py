#!/usr/bin/env python3
"""Read `bundle plan -o json` on stdin and write the part of it a golden should record.

A plan entry's remote_state is the backend's GET payload, copied as is: it carries
backend-computed fields (status, timestamps, sizes, UIDs) and every field the backend
starts or stops returning. Recording it makes a golden fail on backend changes that
leave the CLI's decision (action, new_state, changes) untouched, often on every cloud
and test at once. This drops remote_state from every entry; changes[*].remote still
records the remote value of each field with a change entry.

Also removes the deployment stamp, as nostamp.py does, so it replaces
`nostamp | jq '.plan["KEY"] | del(.remote_state)'`.

Usage:
  $CLI bundle plan -o json | plan_filter.py                  # whole plan
  $CLI bundle plan -o json | plan_filter.py resources.jobs.foo   # one entry

A test that asserts on remote_state itself should not use this.
"""

import argparse
import json
import sys

import nostamp


def filter_plan(data, key=None):
    entries = data.get("plan") if isinstance(data, dict) else None
    if not isinstance(entries, dict):
        raise SystemExit("plan_filter.py: input is not a plan: no 'plan' object")
    for entry in entries.values():
        if isinstance(entry, dict):
            entry.pop("remote_state", None)
    if key is None:
        return data
    if key not in entries:
        raise SystemExit(f"plan_filter.py: no plan entry {key!r}; entries: {sorted(entries)}")
    return entries[key]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("key", nargs="?", help="resource key of the one entry to print, e.g. resources.jobs.foo")
    args = parser.parse_args()
    text = sys.stdin.read()
    try:
        data = json.loads(text)
    except json.JSONDecodeError as ex:
        raise SystemExit(f"plan_filter.py: input is not JSON ({ex}): {text[:200]!r}") from None
    data = filter_plan(data, args.key)
    sys.stdout.write(nostamp.render(data, 2) + "\n")


if __name__ == "__main__":
    main()
