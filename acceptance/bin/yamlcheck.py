#!/usr/bin/env python3

import os
import subprocess
import sys
from pathlib import Path
from difflib import unified_diff


def main():
    current_dir = Path.cwd()
    yaml_files = sorted(current_dir.glob("**/*.yml")) + sorted(current_dir.glob("**/*.yaml"))

    yamlfmt_conf = Path(os.environ["TESTROOT"]) / ".." / "yamlfmt.yml"

    has_changes = False

    for yaml_file in yaml_files:
        # Read original content into memory
        original_content = yaml_file.read_text().splitlines(keepends=True)

        # Run yamlfmt
        subprocess.run(["yamlfmt", f"-conf={yamlfmt_conf}", str(yaml_file)], check=True, capture_output=True)

        # Compare with formatted content
        formatted_content = yaml_file.read_text().splitlines(keepends=True)

        if original_content != formatted_content:
            has_changes = True
            diff = unified_diff(original_content, formatted_content, fromfile=str(yaml_file), tofile=str(yaml_file), lineterm="")
            print("".join(diff))
        else:
            print(f"{yaml_file} OK")

    if has_changes:
        sys.exit(1)


if __name__ == "__main__":
    main()
