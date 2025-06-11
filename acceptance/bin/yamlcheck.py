#!/usr/bin/env python3

import os
import subprocess
import sys
from pathlib import Path
from difflib import unified_diff


def main():
    current_dir = Path.cwd()
    yaml_files = sorted(current_dir.glob("**/*.yml")) + sorted(current_dir.glob("**/*.yaml"))
    if not yaml_files:
        sys.exit("No YAML files found")

    yamlfmt_conf = Path(os.environ["TESTROOT"]) / ".." / "yamlfmt.yml"

    has_changes = False

    for yaml_file in yaml_files:
        original_content = yaml_file.read_text().splitlines(keepends=True)

        subprocess.run(["yamlfmt", f"-conf={yamlfmt_conf}", str(yaml_file)], check=True, capture_output=True)

        formatted_content = yaml_file.read_text().splitlines(keepends=True)

        if original_content != formatted_content:
            has_changes = True
            diff = unified_diff(original_content, formatted_content, fromfile=str(yaml_file), tofile=str(yaml_file), lineterm="")
            print("".join(diff))

    if has_changes:
        sys.exit("UNEXPECTED: YAML formatting issues")


if __name__ == "__main__":
    main()
