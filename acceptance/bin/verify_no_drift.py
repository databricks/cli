#!/usr/bin/env python3
"""
Check that all actions in plan are "skip".
"""

import sys
import json
import pprint


def check_plan(path):
    with open(path) as fobj:
        raw = fobj.read()

    changes_detected = 0

    try:
        data = json.loads(raw)
        for key, value in data["plan"].items():
            action = value.get("action")
            if action != "skip":
                print(f"Unexpected {action=} for {key}")
                changes_detected += 1
    except Exception:
        print(raw, flush=True)
        raise

    if changes_detected:
        print(raw, flush=True)
        sys.exit(10)


def main():
    for path in sys.argv[1:]:
        check_plan(path)


if __name__ == "__main__":
    main()
