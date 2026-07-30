#!/usr/bin/env python3
"""
Read state and add all resource IDs to ACC_REPLS.

Safe to call unconditionally after a deploy: with no -t it processes the state of
every deployed target (both engines) and is a no-op when no state exists, so it
never aborts a script (unlike print_state.py, which errors on ambiguous targets).
"""

import argparse
import glob
import json
import os
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from add_repl import add_repl


def iter_state_files(target):
    if target:
        target_dirs = [f".databricks/bundle/{target}"]
    else:
        target_dirs = glob.glob(".databricks/bundle/*")
    for d in target_dirs:
        for name in (f"{d}/terraform/terraform.tfstate", f"{d}/resources.json"):
            if os.path.exists(name):
                yield name


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
    args = parser.parse_args()

    for filename in iter_state_files(args.target):
        if filename.endswith(".tfstate"):
            it = iter_ids_terraform(filename)
        else:
            it = iter_ids_direct(filename)
        for name, id in it:
            add_repl(id, name.upper() + "_ID")


if __name__ == "__main__":
    main()
