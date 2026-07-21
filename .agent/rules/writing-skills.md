---
description: Conventions for authoring and editing skills under .agent/skills
globs:
  - ".agent/skills/**"
  - ".claude/skills/**"
---

**RULE: Follow the `superpowers:writing-skills` skill when creating or editing a skill.** It defines the authoring discipline (baseline-test the failure first, then write the minimal skill that fixes it). Do not skip it because a change "looks simple".

**RULE: A skill's `description` states only WHEN to use it, never what it does.** Start with "Use when ..." and list concrete triggers. Do not summarize the workflow or steps in the description: agents follow the description as a shortcut and skip reading the body, so a workflow summary there causes them to do the wrong thing.

**RULE: In skill prose, write one sentence per line and do not hard-wrap a sentence across lines.** This keeps copy edits and diffs localized to the sentence that changed instead of reflowing a paragraph. Code blocks, tables, and frontmatter are exempt.

**RULE: Do not use em dashes in skill prose.** Rephrase with a comma, parentheses, a period, or a connecting word.

These apply to skills you author or edit. Do not reformat existing skills that predate these conventions unless you are already changing them for another reason.
