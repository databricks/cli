#!/usr/bin/env python3
"""Run doctests for every Python file in the repo that has them.

Most Python files here are scripts with no doctests, and many can't even be
imported standalone (SDK-generated modules, notebooks using injected globals,
scripts with import-time side effects). So we git grep for files that actually
contain a `>>>` prompt and only run the doctest runner on those; a file with
none produces no tests anyway. --untracked also scans new files not yet
committed while still honoring .gitignore, so generated/ignored files stay out.
Each file runs in its own subprocess with cwd set to the file's directory, so
sibling imports (e.g. gron.py's `from print_requests import ...`) resolve as
they do when the script is invoked normally.
"""

import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent


def main() -> int:
    out = subprocess.check_output(
        ["git", "grep", "-l", "--untracked", "--no-color", "-F", ">>>", "--", "*.py"],
        cwd=ROOT,
        text=True,
    )
    files = sorted(ROOT / line for line in out.splitlines())
    failed = []
    for path in files:
        result = subprocess.run(
            [sys.executable, "-m", "doctest", path.name],
            cwd=path.parent,
        )
        if result.returncode != 0:
            failed.append(path.relative_to(ROOT))

    print(f"\n{len(files)} files with doctests, {len(failed)} failed")
    for rel in failed:
        print(f"FAIL {rel}")
    return 1 if failed else 0


if __name__ == "__main__":
    sys.exit(main())
