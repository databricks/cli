#!/usr/bin/env python3
"""
Check that all actions in plan are "skip".
"""

import json
import sys


def check_plan(path):
    with open(path) as fobj:
        raw = fobj.read()

    # Empty or unparseable output means `bundle plan` itself failed; report that
    # cleanly instead of crashing with a traceback.
    if not raw.strip():
        sys.exit(f"{path}: empty plan output (bundle plan failed)")
    try:
        data = json.loads(raw)
    except json.JSONDecodeError as e:
        sys.exit(f"{path}: invalid plan JSON: {e}\n{raw}")

    changes_detected = 0
    for key, value in data["plan"].items():
        action = value.get("action")
        if action != "skip":
            print(f"Unexpected {action=} for {key}")
            changes_detected += 1

    if changes_detected:
        print(raw, flush=True)
        sys.exit(10)


def main():
    for path in sys.argv[1:]:
        check_plan(path)


if __name__ == "__main__":
    main()
