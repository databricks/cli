#!/usr/bin/env python3
# /// script
# requires-python = ">=3.12"
# ///
"""Keep every agent's config in sync with the canonical .agents/ setup.

The agent instructions live in AGENTS.md and .agents/ (rules, skills); each agent
reads them from its own path via a symlink:

  - CLAUDE.md                      -> AGENTS.md
  - .github/custom-instructions.md -> AGENTS.md
  - .claude/rules, .claude/skills  -> .agents/rules, .agents/skills
  - .cursor/rules/<name>.mdc       -> .agents/rules/<name>.md

Cursor gets a symlink per rule rather than a directory link like Claude, because
it only reads .mdc files and won't follow a link to the .md directory.

This creates any missing symlinks and drops stale Cursor ones, then exits non-zero
if it had to change anything so CI fails on an out-of-sync tree. We can't lean on
CI's `git diff --exit-code` alone: the common case is a missing symlink, which we
create as an untracked file that a plain `git diff` doesn't report.

The standalone .cursor/rules/00-agents-context.mdc bootstrap is not a mirror of a
rule and is left untouched.
"""

import os
import sys

AGENTS_RULES = ".agents/rules"
CURSOR_RULES = ".cursor/rules"

# Symlink path -> target, relative to the symlink's own directory.
FIXED_LINKS = {
    "CLAUDE.md": "AGENTS.md",
    ".github/custom-instructions.md": "../AGENTS.md",
    ".claude/rules": "../.agents/rules",
    ".claude/skills": "../.agents/skills",
}


def ensure_link(path, target):
    if os.path.islink(path) and os.readlink(path) == target:
        return False
    if os.path.lexists(path):
        os.remove(path)
    os.symlink(target, path)
    print(f"Linked {path} -> {target}")
    return True


def main():
    changed = False

    for path, target in FIXED_LINKS.items():
        changed |= ensure_link(path, target)

    stems = sorted(f[:-3] for f in os.listdir(AGENTS_RULES) if f.endswith(".md"))

    # The .mdc lives in .cursor/rules/, so ../../ reaches the repo root.
    for stem in stems:
        changed |= ensure_link(os.path.join(CURSOR_RULES, stem + ".mdc"), f"../../{AGENTS_RULES}/{stem}.md")

    # Drop .mdc symlinks whose rule no longer exists.
    known = {stem + ".mdc" for stem in stems}
    for name in sorted(os.listdir(CURSOR_RULES)):
        path = os.path.join(CURSOR_RULES, name)
        if os.path.islink(path) and name not in known:
            os.remove(path)
            print(f"Removed stale {path}")
            changed = True

    if changed:
        print("agent symlinks were out of sync; they have been fixed, commit the changes", file=sys.stderr)
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
