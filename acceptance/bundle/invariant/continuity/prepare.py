#!/usr/bin/env python3
"""Generate state files for continuity tests.

Usage: python3 prepare.py v0.293.0

Deploys using the specified CLI version for each invariant config against the
local mock server, then saves the resulting state file to:
  continuity/<version>/<config>.state.json

Run this script once and commit the generated files. They are then used by
the continuity/<version>/ test to verify that the current CLI can deploy on
top of state produced by the old version.
"""
import sys
import subprocess
import os
from pathlib import Path


def main():
    if len(sys.argv) != 2:
        print(f"Usage: {sys.argv[0]} <version>", file=sys.stderr)
        sys.exit(1)

    version = sys.argv[1]
    if not version.startswith("v"):
        version = f"v{version}"
    version_num = version[1:]  # strip 'v' prefix for -useversion flag

    repo_root = Path(__file__).resolve().parents[4]

    env = os.environ.copy()
    env["CONTINUITY_VERSION"] = version

    result = subprocess.run(
        [
            "go",
            "test",
            "./acceptance",
            "-run",
            "TestAccept/bundle/invariant/continuity/prepare",
            "-useversion",
            version_num,
            "-forcerun",
            "-v",
        ],
        cwd=repo_root,
        env=env,
    )
    sys.exit(result.returncode)


if __name__ == "__main__":
    main()
