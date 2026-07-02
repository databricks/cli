#!/usr/bin/env python3
"""
Check that all actions in plan are "skip".
"""

import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))

from util import load_plan


def check_plan(path):
    data, raw = load_plan(path)

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
