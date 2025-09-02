#!/usr/bin/env python3

import os
import subprocess
import sys
from pathlib import Path
from difflib import unified_diff


NAME = "yamlfmt"
if sys.platform.startswith("win"):
    NAME += ".exe"


def main():
    current_dir = Path.cwd()
    yaml_files = sorted(current_dir.glob("**/*.yml")) + sorted(current_dir.glob("**/*.yaml"))
    if not yaml_files:
        sys.exit("No YAML files found")

    yamlfmt_conf = Path(os.environ["TESTROOT"]) / ".." / "yamlfmt.yml"
    yamlfmt = Path(os.environ["TESTROOT"]) / ".." / "tools" / NAME

    has_changes = []

    for yaml_file in yaml_files:
        original_content = yaml_file.read_text().splitlines(keepends=True)

        subprocess.run([yamlfmt, f"-conf={yamlfmt_conf}", str(yaml_file)], check=True, capture_output=True)

        formatted_content = yaml_file.read_text().splitlines(keepends=True)

        if original_content != formatted_content:
            has_changes.append(str(yaml_file))
            # Add $ markers for trailing whitespace
            original_with_markers = [line.rstrip("\n") + ("$" if line.rstrip() != line.rstrip("\n") else "") + "\n" for line in original_content]
            formatted_with_markers = [line.rstrip("\n") + ("$" if line.rstrip() != line.rstrip("\n") else "") + "\n" for line in formatted_content]
            diff = unified_diff(original_with_markers, formatted_with_markers, fromfile=str(yaml_file), tofile=str(yaml_file), lineterm="")
            print("".join(diff))

    if has_changes:
        sys.exit("UNEXPECTED: YAML formatting issues in " + " ".join(has_changes))


if __name__ == "__main__":
    main()
