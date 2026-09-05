#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Keep .cursor/rules/*.mdc in sync with the canonical .agents/rules/*.md rules.

The agent instructions live in AGENTS.md and .agents/ (rules, skills). Most agents
reach them through a single committed symlink that needs no per-rule upkeep:

  - CLAUDE.md                      -> AGENTS.md
  - .github/custom-instructions.md -> AGENTS.md
  - .claude/rules, .claude/skills  -> .agents/rules, .agents/skills  (directory links)

Cursor is the exception: it only reads .mdc files and won't follow a directory
link, so every rule needs its own .cursor/rules/<name>.mdc pointing back at the .md.
This creates the missing ones and drops stale ones, then exits non-zero if it had
to change anything so CI fails on an out-of-sync tree: a created symlink is
untracked, so CI's `git diff --exit-code` gate alone would let a missing one pass.

The standalone .cursor/rules/00-agents-context.mdc bootstrap is not a mirror of a
rule and is left untouched.
"""

import os
import sys

AGENTS_RULES = ".agents/rules"
CURSOR_RULES = ".cursor/rules"


def main():
    changed = False

    stems = sorted(f[:-3] for f in os.listdir(AGENTS_RULES) if f.endswith(".md"))

    # Every rule needs a .mdc symlink. The .mdc lives in .cursor/rules/, so ../../
    # reaches the repo root.
    for stem in stems:
        mdc = os.path.join(CURSOR_RULES, stem + ".mdc")
        want = f"../../{AGENTS_RULES}/{stem}.md"
        if os.path.islink(mdc) and os.readlink(mdc) == want:
            continue
        if os.path.lexists(mdc):
            os.remove(mdc)
        os.symlink(want, mdc)
        print(f"Linked {mdc} -> {want}")
        changed = True

    # Drop .mdc symlinks whose rule no longer exists.
    known = {stem + ".mdc" for stem in stems}
    for name in sorted(os.listdir(CURSOR_RULES)):
        path = os.path.join(CURSOR_RULES, name)
        if os.path.islink(path) and name not in known:
            os.remove(path)
            print(f"Removed stale {path}")
            changed = True

    if changed:
        print("Cursor rule symlinks were out of sync; they have been fixed, commit the changes", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
