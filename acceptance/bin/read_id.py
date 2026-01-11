#!/usr/bin/env python3
"""
Print id of the resource from the state. Update ACC_REPLS for a given ID.

 Example: read_id.py foo
 Output job_id, e.g. "5555" and update ACC_REPLS with record "5555:FOO_ID"

Usage: <group> <name> [attr...]
"""

import sys
import json
import argparse
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from print_state import get_state_file
from add_repl import add_repl


def get_id_terraform(filename, name):
    raw = open(filename).read()
    data = json.loads(raw)
    available = []
    for r in data["resources"]:
        r_name = r["name"]
        available.append(r_name)
        if r_name == name:
            for inst in r["instances"]:
                attribute_values = inst.get("attributes") or {}
                return attribute_values.get("id")

    print(f"Cannot find resource with {name=}. Available: {available}", file=sys.stderr)


def get_id_direct(filename, name):
    raw = open(filename).read()
    data = json.loads(raw)
    state_map = data["state"]

    for key, value in state_map.items():
        if key.split(".")[2] == name:
            return value.get("__id__")

    print(f"Cannot find resource with {name=}. Available: {list(state_map.keys())}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    parser.add_argument("--backup", action="store_true")
    parser.add_argument("name")
    args = parser.parse_args()

    filename = get_state_file(args.target, args.backup)
    if filename.endswith(".tfstate"):
        id = get_id_terraform(filename, args.name)
    else:
        id = get_id_direct(filename, args.name)

    if id:
        print(id)
        add_repl(str(id), args.name.upper() + "_ID")


if __name__ == "__main__":
    main()
