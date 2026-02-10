#!/usr/bin/env python3
"""
Read state and add all resource IDs to ACC_REPLS.
"""

import sys
import json
import argparse
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from print_state import get_state_file
from add_repl import add_repl


def iter_ids_terraform(filename):
    raw = open(filename).read()
    data = json.loads(raw)
    available = []
    for r in data["resources"]:
        r_name = r["name"]
        available.append(r_name)
        for inst in r["instances"]:
            attribute_values = inst.get("attributes") or {}
            id = attribute_values.get("id")
            yield r_name, id


def iter_ids_direct(filename):
    raw = open(filename).read()
    data = json.loads(raw)
    state_map = data["state"]

    for key, value in state_map.items():
        name = key.split(".")[2]
        id = value.get("__id__")
        if id:
            yield name, id


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    parser.add_argument("--backup", action="store_true")
    args = parser.parse_args()

    filename = get_state_file(args.target, args.backup)
    if filename.endswith(".tfstate"):
        it = iter_ids_terraform(filename)
    else:
        it = iter_ids_direct(filename)

    for name, id in it:
        add_repl(id, name.upper() + "_ID")


if __name__ == "__main__":
    main()
