#!/usr/bin/env python3
"""
Print resources state from default target.

Note, this intentionally has no logic on guessing what is the right state file (e.g. via DATABRICKS_BUNDLE_ENGINE),
the goal is to record all states that are available.
"""

import os
import glob
import argparse


def print_file(filename):
    data = open(filename).read()
    print(data, end="")
    if not data.endswith("\n"):
        print()


def get_state_files(target, backup):
    default_target_dir = ".databricks/bundle/default"

    if target:
        target_dir = f".databricks/bundle/{target}"
        if not os.path.exists(target_dir):
            raise SystemExit(f"Invalid target {target!r}: {target_dir} does not exist")
    elif os.path.exists(default_target_dir):
        target_dir = default_target_dir
    else:
        targets = glob.glob(".databricks/bundle/*")
        if not targets:
            return
        targets = [os.path.basename(x) for x in targets]
        if len(targets) > 1:
            raise SystemExit("Many targets found, specify one to use with -t: " + ", ".join(sorted(targets)))
        target_dir = ".databricks/bundle/" + targets[0]

    result = []

    if backup:
        result.append(f"{target_dir}/terraform/terraform.tfstate.backup")
    else:
        result.append(f"{target_dir}/terraform/terraform.tfstate")
    result.append(f"{target_dir}/resources.json")
    return result


def get_state_file(target, backup):
    result = get_state_files(target, backup)
    filtered = [x for x in result if os.path.exists(x)]
    return filtered[0] if filtered else result[0]


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    parser.add_argument("--backup", action="store_true")
    args = parser.parse_args()

    for filename in get_state_files(args.target, args.backup):
        if os.path.exists(filename):
            print_file(filename)


if __name__ == "__main__":
    main()
