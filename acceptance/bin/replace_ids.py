#!/usr/bin/env python3
"""
Read state and add all resource IDs to ACC_REPLS.
"""

import argparse
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from add_repl import add_repl
from print_state import get_resources, get_state_file


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


def iter_ids_recorded(target):
    for key, value in get_resources(target).items():
        if value["id"]:
            yield key.split(".")[1], value["id"]


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

    if os.environ.get("DATABRICKS_BUNDLE_DEPLOYMENT_HISTORY") == "true":
        it = iter_ids_recorded(args.target)
    else:
        filename = get_state_file(args.target, args.backup)
        if filename.endswith(".tfstate"):
            it = iter_ids_terraform(filename)
        else:
            it = iter_ids_direct(filename)

    for name, id in it:
        add_repl(id, name.upper() + "_ID")


if __name__ == "__main__":
    main()
