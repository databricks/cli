#!/usr/bin/env python3
"""Run go test -update for all packages that use libs/testdiff."""

import os
import subprocess
import sys

from pathlib import Path


def main():
    result = subprocess.run(
        ["git", "grep", "-l", "libs/testdiff", "--", "**/*_test.go"],
        capture_output=False,
        stdout=subprocess.PIPE,
        text=True,
        check=True,
    )

    dirs = {str(Path(f).parent) for f in result.stdout.splitlines() if f}
    dirs.discard("acceptance")
    packages = sorted(f"./{d}" for d in dirs)

    if not packages:
        print("No packages found.", file=sys.stderr)
        sys.exit(1)

    cmd = ["go", "test", *packages, "-update"]
    print(" ".join(cmd), file=sys.stderr, flush=True)
    os.execvp(cmd[0], cmd)


if __name__ == "__main__":
    main()
