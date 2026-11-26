#!/usr/bin/env python3
"""
Run `bundle generate` for every top-level resource recorded in the deployment state.

    generate_from_state.py [-t TARGET] --bundle-dir DIR

State is read from the current directory; each resource is generated into the bundle at DIR
under its own key with `--existing-id <id>`, so downloaded files land inside that bundle.
Prints nothing on success so the output is identical across configs; on failure prints the
command and its output and exits non-zero.
"""

import argparse
import json
import subprocess
import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).parent))
from print_state import CLI, get_state_file


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("-t", "--target")
    parser.add_argument("--bundle-dir", required=True)
    args = parser.parse_args()

    state = json.loads(Path(get_state_file(args.target, False)).read_text())["state"]
    for key in sorted(state):
        parts = key.split(".")
        # Sub-resources such as resources.jobs.foo.permissions are generated with their parent.
        if len(parts) != 3:
            continue
        _, group, name = parts
        cmd = [CLI, "bundle", "generate", group, "--existing-id", state[key]["__id__"], "--key", name]
        result = subprocess.run(
            cmd, cwd=args.bundle_dir, stdout=subprocess.PIPE, stderr=subprocess.STDOUT, encoding="utf-8"
        )
        if result.returncode != 0:
            print(f"{' '.join(cmd[1:])} failed with exit code {result.returncode}:\n{result.stdout}", flush=True)
            sys.exit(result.returncode)


if __name__ == "__main__":
    main()
