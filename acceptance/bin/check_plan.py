#!/usr/bin/env python3
import sys
import json


def check_plan(path):
    with open(path) as fobj:
        data = fobj.read()

    changes_detected = 0

    data = json.loads(data)
    for key, value in data["plan"].items():
        if value["action"] != "skip":
            print("Unexpected action in", key, value["action"])
            pprint.pprint(value)
            print()
            changes_detected += 1

    if changes_detected:
        sys.exit(10)


def main():
    for path in sys.argv[1:]:
        check_plan(path)


if __name__ == "__main__":
    main()
