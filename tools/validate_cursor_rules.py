#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Keep .cursor/rules/*.mdc symlinks in sync with .agents/rules/*.md.

The canonical rules live in .agents/rules/<name>.md; Cursor reads them from
.cursor/rules/<name>.mdc, so each rule needs a .mdc symlink pointing back at its
.md. This validates that every rule has a correct symlink and that no symlink is
left dangling. Run with --fix to create missing symlinks and drop stale ones.

Non-symlink .mdc files (e.g. the standalone 00-agents-context.mdc bootstrap) are
not mirrors of a rule and are left untouched.
"""

import os
import sys

AGENTS_RULES = ".agents/rules"
CURSOR_RULES = ".cursor/rules"


def link_target(stem):
    # The .mdc lives in .cursor/rules/, so ../../ reaches the repo root.
    return f"../../{AGENTS_RULES}/{stem}.md"


def main():
    fix = "--fix" in sys.argv

    stems = sorted(f[:-3] for f in os.listdir(AGENTS_RULES) if f.endswith(".md"))
    problems = []

    # Every rule must have a .mdc symlink pointing at its .md.
    for stem in stems:
        mdc = os.path.join(CURSOR_RULES, stem + ".mdc")
        want = link_target(stem)
        have = os.readlink(mdc) if os.path.islink(mdc) else None
        if have == want:
            continue
        if fix:
            if os.path.lexists(mdc):
                os.remove(mdc)
            os.symlink(want, mdc)
            print(f"Linked {mdc} -> {want}")
        else:
            problems.append(f"{mdc}: missing or wrong symlink, expected -> {want}")

    # No .mdc symlink may point at a rule that no longer exists.
    known = {stem + ".mdc" for stem in stems}
    for name in sorted(os.listdir(CURSOR_RULES)):
        path = os.path.join(CURSOR_RULES, name)
        if not os.path.islink(path) or name in known:
            continue
        if fix:
            os.remove(path)
            print(f"Removed stale {path}")
        else:
            problems.append(f"{path}: stale symlink, no matching {AGENTS_RULES}/ rule")

    if problems:
        print("\n".join(problems))
        print(
            f"\n{len(problems)} problem(s). Run: ./tools/validate_cursor_rules.py --fix",
            file=sys.stderr,
        )
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
