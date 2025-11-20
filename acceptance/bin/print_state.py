#!/usr/bin/env python3
"""
Print resources state from default target.

Note, this intentionally has no logic on guessing what is the right state file (e.g. via DATABRICKS_BUNDLE_ENGINE),
the goal is to record all states that are available.
"""

import os
import argparse


def write(filename):
    data = open(filename).read()
    print(data, end="")
    if not data.endswith("\n"):
        print()


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target", default="default")
    args = parser.parse_args()

    if args.target:
        target_dir = f".databricks/bundle/{args.target}"
        if not os.path.exists(target_dir):
            raise SystemExit(f"Invalid target {args.target!r}: {target_dir} does not exist")

    filename = f".databricks/bundle/{args.target}/terraform/terraform.tfstate"
    if os.path.exists(filename):
        write(filename)

    filename = f".databricks/bundle/{args.target}/resources.json"
    if os.path.exists(filename):
        write(filename)


if __name__ == "__main__":
    main()
