#!/usr/bin/env python3
"""Refresh the vendored mlops-stacks acceptance template to upstream HEAD.

The local test leg inits from the vendored copy; the cloud leg clones upstream.
Only already-vendored files are refreshed, which preserves the pruning.
"""

import os
import shutil
import subprocess
import sys
import tempfile

REPO_URL = "https://github.com/databricks/mlops-stacks"

# Vendored template root == upstream repo root.
TEMPLATE_DIR = "acceptance/bundle/deploy/mlops-stacks/template"
REVISION_FILE = "acceptance/bundle/deploy/mlops-stacks/template.REVISION"

# Upstream's project dir name is an illegal Go module path, so we vendor it renamed.
VENDORED_PROJECT_DIR = "{{.input_project_name}}"
UPSTREAM_PROJECT_DIR = "{{template `project_name_alphanumeric_underscore` .}}"


def repo_root():
    return os.path.normpath(os.path.join(os.path.dirname(os.path.abspath(sys.argv[0])), ".."))


def vendored_to_upstream(rel_path):
    """Map a path relative to the vendored template root to its upstream path."""
    parts = rel_path.split(os.sep)
    return os.path.join(*[UPSTREAM_PROJECT_DIR if p == VENDORED_PROJECT_DIR else p for p in parts])


def main():
    root = repo_root()
    os.chdir(root)

    template_dir = os.path.join(root, TEMPLATE_DIR)
    if not os.path.isdir(template_dir):
        sys.exit(f"vendored template dir not found: {TEMPLATE_DIR}")

    vendored = []
    for dirpath, _, filenames in os.walk(template_dir):
        for name in filenames:
            full = os.path.join(dirpath, name)
            vendored.append(os.path.relpath(full, template_dir))
    vendored.sort()

    with tempfile.TemporaryDirectory(prefix="mlops-stacks-") as clone_dir:
        subprocess.run(["git", "clone", "--depth", "1", "--quiet", REPO_URL, clone_dir], check=True)
        sha = subprocess.run(
            ["git", "-C", clone_dir, "rev-parse", "HEAD"],
            check=True,
            capture_output=True,
            text=True,
        ).stdout.strip()

        # Resolve all sources first so a restructured upstream leaves the copy untouched.
        sources = {rel: os.path.join(clone_dir, vendored_to_upstream(rel)) for rel in vendored}
        missing = [rel for rel, src in sources.items() if not os.path.isfile(src)]
        if missing:
            listing = "\n".join(f"  {m}" for m in missing)
            sys.exit(
                "upstream no longer provides these vendored files (it may have "
                f"restructured); reconcile manually:\n{listing}"
            )

        for rel, src in sources.items():
            shutil.copyfile(src, os.path.join(template_dir, rel))

    with open(os.path.join(root, REVISION_FILE), "w") as f:
        f.write(sha + "\n")

    print(f"Refreshed {len(vendored)} files from {REPO_URL} @ {sha}")
    print("Review the diff, then update the golden output with:")
    print("  go test ./acceptance -run TestAccept/bundle/deploy/mlops-stacks -update")


if __name__ == "__main__":
    main()
