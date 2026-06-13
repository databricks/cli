#!/usr/bin/env python3
"""Sweep leaked acceptance-test resources by name prefix.

Lists (and with --delete, deletes) warehouses, pipelines and jobs whose names
start with the given prefix, e.g. the per-run prefix "ci-<GITHUB_RUN_ID>-" that
the acceptance harness embeds into $UNIQUE_NAME on CI cloud runs.

Authentication is taken from the environment (DATABRICKS_HOST, DATABRICKS_TOKEN
or any other auth supported by the databricks CLI).

Usage:
    tools/sweep_test_resources.py ci-15799017600-           # dry run: list only
    tools/sweep_test_resources.py ci-15799017600- --delete  # delete matches
"""

import argparse
import json
import subprocess
import sys


def run_json(*args):
    out = subprocess.check_output(["databricks", *args, "--output", "json"], text=True)
    return json.loads(out) if out.strip() else []


def sweep(kind, items, name_of, id_of, delete_args, prefix, delete):
    failures = 0
    for item in items:
        name = name_of(item) or ""
        if not name.startswith(prefix):
            continue
        res_id = str(id_of(item))
        print(f"{kind}\t{res_id}\t{name}")
        if delete:
            try:
                subprocess.check_call(["databricks", *delete_args, res_id])
            except subprocess.CalledProcessError as e:
                print(f"failed to delete {kind} {res_id}: {e}", file=sys.stderr)
                failures += 1
    return failures


def main():
    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("prefix", help="resource name prefix, e.g. ci-<GITHUB_RUN_ID>-")
    parser.add_argument("--delete", action="store_true", help="delete matches (default: list only)")
    args = parser.parse_args()

    if not args.prefix:
        parser.error("prefix must not be empty")

    failures = 0
    failures += sweep(
        "warehouse",
        run_json("warehouses", "list"),
        lambda w: w.get("name"),
        lambda w: w.get("id"),
        ["warehouses", "delete"],
        args.prefix,
        args.delete,
    )
    failures += sweep(
        "pipeline",
        run_json("pipelines", "list-pipelines"),
        lambda p: p.get("name"),
        lambda p: p.get("pipeline_id"),
        ["pipelines", "delete"],
        args.prefix,
        args.delete,
    )
    failures += sweep(
        "job",
        run_json("jobs", "list"),
        lambda j: j.get("settings", {}).get("name"),
        lambda j: j.get("job_id"),
        ["jobs", "delete"],
        args.prefix,
        args.delete,
    )
    return 1 if failures else 0


if __name__ == "__main__":
    sys.exit(main())
