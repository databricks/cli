#!/usr/bin/env python3
"""
Check that a `bundle plan -o json` shows an expected action, for invariants beyond
no-drift.

  verify_plan_action.py PATH update   every changed resource is an in-place update
                                       (not a recreate) and at least one changed
  verify_plan_action.py PATH create    every resource is a create (e.g. a re-plan
                                       after destroy must recreate everything)

Action vocabulary mirrors bundle/deployplan/action.go.
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from util import load_plan

# update_id/resize keep the resource (no recreate), so they count as in-place updates.
ALLOWED = {
    "update": {"update", "update_id", "resize"},
    "create": {"create"},
}
# After a destroy, a "skip" means the resource survived (orphaned state), so skip is
# only tolerated for the update check, where unrelated siblings may be unchanged.
SKIP_OK = {"update": True, "create": False}


def main():
    path, expected = sys.argv[1], sys.argv[2]
    allowed = ALLOWED[expected]
    skip_ok = SKIP_OK[expected]

    data, raw = load_plan(path)

    matched = 0
    bad = 0
    for key, value in data["plan"].items():
        action = value.get("action")
        if action == "skip" and skip_ok:
            continue
        if action in allowed:
            matched += 1
        else:
            print(f"Unexpected {action=} for {key} (expected {expected})")
            bad += 1

    if matched == 0:
        print(f"plan shows no {expected} action; expected at least one")
        bad += 1

    if bad:
        print(raw, flush=True)
        sys.exit(10)


if __name__ == "__main__":
    main()
