#!/usr/bin/env python3
"""
Push the replayed terraform.tfstate to the bundle's remote state directory, standing in for the
remote state a real terraform deploy leaves behind.

replay_tfstate.py reconstructs the backend resources and writes the LOCAL terraform.tfstate, but a
real terraform deploy also pushes that state to the workspace. A migrate step that inspects the
remote state dir (workspace list) or backs it up / deletes it (the tfstate-backup path) needs the
remote copy to exist, so those tests run this right after replay_tfstate.py. Tests that only look
at the local state (or use `bundle debug states`) do not need it.

Usage: push_remote_tfstate.py [-t TARGET]
"""

import argparse
import os
import subprocess
import sys

import print_state

CLI = os.environ["CLI"]


def run(*args):
    r = subprocess.run([CLI, *args], stdout=subprocess.PIPE, stderr=subprocess.STDOUT, encoding="utf-8")
    if r.returncode != 0:
        sys.exit(f"push_remote_tfstate.py: {' '.join(args)} failed:\n{r.stdout}")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target", default="default")
    args = parser.parse_args()

    local = print_state.get_state_file(args.target, False)
    state_dir = print_state.get_remote_state_path(args.target)

    # workspace import does not create parents (see the fake server's requireParentDirectory), so
    # mkdirs the state dir first, mirroring how the real deploy pushes state.
    run("workspace", "mkdirs", state_dir)
    run("workspace", "import", f"{state_dir}/terraform.tfstate", "--file", local, "--format", "AUTO", "--overwrite")


if __name__ == "__main__":
    main()
