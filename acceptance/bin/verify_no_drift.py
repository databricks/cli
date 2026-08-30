#!/usr/bin/env python3
"""
Check the actions in a JSON plan.

By default every action must be "skip" -- i.e. a deploy left no drift behind.

--expect-change instead requires a planned change for one field, with the shape the
caller created. "local" means the config moved away from state and remote
(old != new); "remote" means only the remote moved (old == new, remote differs).
Checking the shape and not just the action matters: reverting a config produces a
local change that looks like drift detection from the action alone.
"""

import argparse
import json
import sys

SKIP = "skip"
LOCAL = "local"
REMOTE = "remote"


class Missing:
    """Stands in for an omitted old/new/remote value; the plan omits empty ones."""

    def __repr__(self):
        return "absent"


MISSING = Missing()


def load_plan(path):
    with open(path) as fobj:
        raw = fobj.read()
    try:
        return raw, json.loads(raw)["plan"]
    except Exception:
        print(raw, flush=True)
        raise


def fail(raw, message):
    print(message)
    print(raw, flush=True)
    sys.exit(10)


def check_no_drift(path):
    raw, plan = load_plan(path)

    changes_detected = 0
    for key, value in plan.items():
        action = value.get("action")
        if action != SKIP:
            print(f"Unexpected {action=} for {key}")
            changes_detected += 1

    if changes_detected:
        print(raw, flush=True)
        sys.exit(10)


def find_change(plan, field):
    for key, value in plan.items():
        change = value.get("changes", {}).get(field)
        if change is not None:
            return key, change
    return None, None


def check_expected_change(path, field, shape):
    raw, plan = load_plan(path)

    key, change = find_change(plan, field)
    if change is None:
        fail(raw, f"No planned change for field {field!r}")

    action = change.get("action")
    if action == SKIP:
        fail(raw, f"Change for {key}.{field} was skipped ({change.get('reason')!r})")

    old = change.get("old", MISSING)
    new = change.get("new", MISSING)
    remote = change.get("remote", MISSING)

    if shape == LOCAL and old == new:
        fail(raw, f"Change for {key}.{field} is not a local change: old == new == {new!r}")
    if shape == REMOTE:
        if old != new:
            fail(raw, f"Change for {key}.{field} is not remote-only: {old=} != {new=}")
        if remote == new:
            fail(raw, f"Change for {key}.{field} has no remote drift: remote == new == {new!r}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("filenames", nargs="+")
    parser.add_argument("--expect-change", metavar="FIELD")
    parser.add_argument("--shape", choices=[LOCAL, REMOTE])
    args = parser.parse_args()

    if bool(args.expect_change) != bool(args.shape):
        parser.error("--expect-change and --shape must be given together")

    for path in args.filenames:
        if args.expect_change:
            check_expected_change(path, args.expect_change, args.shape)
        else:
            check_no_drift(path)


if __name__ == "__main__":
    main()
