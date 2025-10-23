#!/usr/bin/env python3

import json
import subprocess
import sys
from pathlib import Path


def check_user_agent(file_path):
    """Check if User-Agent header contains correct engine type based on filename."""
    filename = file_path.name

    if ".terraform." in filename:
        expected = "engine/terraform"
    elif ".direct" in filename:
        expected = "engine/direct"
    else:
        return 0, 0

    # Use print_requests.py to convert to one-line-per-JSON format
    result = subprocess.run(
        ["print_requests.py", "--get", "--oneline", "--fname", str(file_path), "--keep"], capture_output=True, text=True
    )

    found = 0
    not_found = 0

    for line in result.stdout.strip().split("\n"):
        if not line:
            continue

        try:
            data = json.loads(line)
        except Exception:
            print(f"Failed to parse: {line!r}")
            raise
        user_agent = data["headers"]["User-Agent"][0]

        if expected in user_agent:
            found += 1
        else:
            not_found += 1
            print(f"{file_path}: Expected '{expected}' in User-Agent: {user_agent!r} path={data.get('path')}")

    return found, not_found


def main():
    """Walk file tree and validate JSON files."""
    all_passed = True

    cwd = Path.cwd()

    for fname in cwd.rglob("*.json"):
        fname = fname.relative_to(cwd)
        found, not_found = check_user_agent(fname)
        print(f"{found}/{found + not_found} {fname}\n")
        if not_found:
            all_passed = False

    if not all_passed:
        sys.exit(1)


if __name__ == "__main__":
    main()
