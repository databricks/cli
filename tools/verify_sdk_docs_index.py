#!/usr/bin/env python3
"""
Verify that sdk_docs_index.json is up to date.

Usage:
    python3 tools/verify_sdk_docs_index.py

This script regenerates the SDK docs index and compares it with the committed
version to detect if someone forgot to run `make sdk-docs-index` after
changing the SDK version.

Exit codes:
    0 - Index is up to date
    1 - Index is out of date (needs regeneration)
    2 - Error during verification
"""

import json
import subprocess
import sys
import tempfile
from pathlib import Path


def main():
    script_dir = Path(__file__).parent
    project_root = script_dir.parent
    index_path = project_root / "experimental/aitools/lib/providers/sdkdocs/sdk_docs_index.json"

    if not index_path.exists():
        print(f"ERROR: SDK docs index not found at {index_path}")
        sys.exit(2)

    # Read current index
    with open(index_path) as f:
        current_index = json.load(f)

    # Regenerate index to temp directory
    with tempfile.TemporaryDirectory() as tmp_dir:
        tmp_path = Path(tmp_dir)

        result = subprocess.run(
            ["go", "run", "./tools/gen_sdk_docs_index.go", "-output", str(tmp_path) + "/"],
            cwd=project_root,
            capture_output=True,
            text=True,
        )

        if result.returncode != 0:
            print(f"ERROR: Failed to regenerate SDK docs index")
            print(result.stderr)
            sys.exit(2)

        # The generator creates the file with a fixed name
        generated_path = tmp_path / "sdk_docs_index.json"
        if not generated_path.exists():
            print(f"ERROR: Generated index not found at {generated_path}")
            print(f"Generator output: {result.stdout}")
            sys.exit(2)

        with open(generated_path) as f:
            new_index = json.load(f)

        # Compare indexes (ignoring generated_at timestamp)
        current_comparable = {k: v for k, v in current_index.items() if k != "generated_at"}
        new_comparable = {k: v for k, v in new_index.items() if k != "generated_at"}

        if current_comparable == new_comparable:
            print("SDK docs index is up to date.")
            sys.exit(0)

        # Find differences
        print("SDK docs index is OUT OF DATE!")
        print("")

        if current_index.get("sdk_version") != new_index.get("sdk_version"):
            print(f"  SDK version changed: {current_index.get('sdk_version')} -> {new_index.get('sdk_version')}")

        current_services = set(current_index.get("services", {}).keys())
        new_services = set(new_index.get("services", {}).keys())
        if current_services != new_services:
            added = new_services - current_services
            removed = current_services - new_services
            if added:
                print(f"  Services added: {added}")
            if removed:
                print(f"  Services removed: {removed}")

        current_types = len(current_index.get("types", {}))
        new_types = len(new_index.get("types", {}))
        if current_types != new_types:
            print(f"  Types count changed: {current_types} -> {new_types}")

        current_enums = len(current_index.get("enums", {}))
        new_enums = len(new_index.get("enums", {}))
        if current_enums != new_enums:
            print(f"  Enums count changed: {current_enums} -> {new_enums}")

        print("")
        print("Run `make sdk-docs-index` to update the index.")
        sys.exit(1)


if __name__ == "__main__":
    main()
