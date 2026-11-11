#!/usr/bin/env python3
"""
Check that a plan applies at least one "update" and nothing more drastic.

Used by the field_removal invariant test after dropping an optional field: the
removal must be observable as an in-place update, never ignored (all "skip") and
never a create/delete/recreate.
"""

import json
import sys


def check_plan(path):
    with open(path) as fobj:
        raw = fobj.read()

    updates = 0
    unexpected = 0

    try:
        data = json.loads(raw)
        for key, value in data["plan"].items():
            action = value.get("action")
            if action == "update":
                updates += 1
            elif action != "skip":
                print(f"Unexpected {action=} for {key}")
                unexpected += 1
    except Exception:
        print(raw, flush=True)
        raise

    if not updates:
        print("Expected at least one update action, found none")
        unexpected += 1

    if unexpected:
        print(raw, flush=True)
        sys.exit(10)


def main():
    for path in sys.argv[1:]:
        check_plan(path)


if __name__ == "__main__":
    main()
