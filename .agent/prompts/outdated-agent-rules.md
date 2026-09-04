Review whether the agent instructions in `.agent/`, `.claude/`, `.cursor/`, `AGENTS.md`, and `CLAUDE.md` are still up to date and correct. Report findings; do not fix without confirmation.

Check for:

- **Missing symlinks.** Every `.agent/rules/*.md` should have a matching `.cursor/rules/*.mdc` symlink pointing at it. Run `ls -la .agent/rules/ .cursor/rules/` and diff the two listings.
- **Stale paths.** Files referenced in rules may have moved. Pick a sample of paths quoted in `.agent/rules/*.md`, `AGENTS.md`, and `CLAUDE.md` and verify each still exists.
- **Stale commands.** Make targets and tool invocations referenced in `.agent/skills/pr-checklist/SKILL.md`, `AGENTS.md`, `CLAUDE.md`, and `.claude/settings.json` should still match the current `Makefile`. Compare the documented behavior of each `make <target>` against its actual recipe (e.g. `make checks` is documented as "tidy, whitespace, links" but the recipe is `tidy ws links deadcode`).
- **Permission allowlist drift.** `.claude/settings.json` allows commands that may have been renamed or removed. Check each `Bash(...)` entry against currently-used tooling and the Makefile.
- **Rule drift vs. code.** Rules that name specific files, packages, helpers, or APIs (e.g. `libs/cmdio`, `apierr.ErrResourceDoesNotExist`) should be grepped to confirm they still exist with that name and signature.
- **Duplicate or contradictory guidance.** When the same rule appears in multiple places (e.g. `AGENTS.md` and a more specific `.agent/rules/*.md`), confirm they don't disagree.

Report each issue with the file:line and a one-line description. Do not edit files without the user's go-ahead.
