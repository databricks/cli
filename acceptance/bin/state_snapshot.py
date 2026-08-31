#!/usr/bin/env python3
"""
Snapshot and restore the deployment state file.

Restoring a snapshot makes an update the CLI performed itself look like an
out-of-band remote change: state says one thing, the remote says another.

The engine is not hardcoded here -- print_state.get_state_file() resolves the
target directory and picks resources.json (direct) or terraform.tfstate
(terraform), so both engines go through the same helper.
"""

import argparse
import json
import os
import shutil
import sys

sys.path.insert(0, os.path.dirname(__file__))
import print_state

# Written during a deploy and merged into the state file at the end of it. A WAL
# left over from a crashed deploy would be replayed on top of a restored
# snapshot, undoing the restore, so drop it alongside.
WAL_SUFFIX = ".wal"


def snapshot_path(state_path):
    return state_path + ".snapshot"


def save(target):
    state_path = print_state.get_state_file(target, backup=False)
    if not os.path.exists(state_path):
        sys.exit(f"cannot snapshot {state_path}: file does not exist")
    shutil.copyfile(state_path, snapshot_path(state_path))


def read_serial(path):
    with open(path) as fobj:
        return json.load(fobj)["serial"]


def restore(target):
    state_path = print_state.get_state_file(target, backup=False)
    snapshot = snapshot_path(state_path)
    if not os.path.exists(snapshot):
        sys.exit(f"cannot restore {state_path}: {snapshot} does not exist")

    with open(snapshot) as fobj:
        data = json.load(fobj)

    # The deploy that ran since the snapshot bumped the serial and pushed the state to
    # the workspace. statemgmt.readStates picks whichever copy has the highest serial, so
    # a snapshot restored at its original serial loses to that remote copy and the
    # restore silently does nothing. Re-stamping it past the current serial makes the
    # restored state authoritative again.
    data["serial"] = read_serial(state_path) + 1

    with open(state_path, "w") as fobj:
        json.dump(data, fobj)

    wal = state_path + WAL_SUFFIX
    if os.path.exists(wal):
        os.remove(wal)


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("action", choices=["save", "restore"])
    parser.add_argument("-t", "--target")
    args = parser.parse_args()

    if args.action == "save":
        save(args.target)
    else:
        restore(args.target)


if __name__ == "__main__":
    main()
