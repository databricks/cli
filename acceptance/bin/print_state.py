#!/usr/bin/env python3
"""
Print resources state from default target.

Note, this intentionally has no logic on guessing what is the right state file (e.g. via DATABRICKS_BUNDLE_ENGINE),
the goal is to record all states that are available.
"""

import argparse
import glob
import json
import os


def records_deployment_history():
    return os.environ.get("DATABRICKS_BUNDLE_RECORD_DEPLOYMENT_HISTORY") == "true"


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


def print_recorded_state(filename, target):
    """Print the state file with its resources filled in from the deployment metadata service.

    While recording, the file itself carries only the header - the service holds the resources - so
    printing it raw would show an empty state and differ from the same test's non-recording run.
    """
    # Imported here rather than at module level: dms_resources reads get_state_file from this module.
    from dms_resources import get_resources

    data = json.loads(open(filename).read())
    state = {}
    for key, value in sorted(get_resources(target).items()):
        entry = {"__id__": value["id"], "state": value["state"]}
        if value["depends_on"]:
            entry["depends_on"] = value["depends_on"]
        state[f"resources.{key}"] = entry
    data["state"] = state
    print(json.dumps(data, indent=1))


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    parser.add_argument("--backup", action="store_true")
    args = parser.parse_args()

    for filename in get_state_files(args.target, args.backup):
        if not os.path.exists(filename):
            continue
        if filename.endswith("resources.json") and records_deployment_history():
            print_recorded_state(filename, args.target)
        else:
            print_file(filename)


if __name__ == "__main__":
    main()
