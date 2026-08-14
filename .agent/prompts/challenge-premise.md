Use this prompt to red-team a change that rests on a load-bearing assumption — a claim about how a tool, API, the repo, or the codebase behaves. Run it alongside the normal correctness review, not instead of it.

Your job is to argue this PR should NOT be merged. Assume the implementation is correct; do not re-check it. Instead:

- State the change's load-bearing assumption explicitly: the one claim that, if false, makes the change unnecessary or wrong.
- Try to disprove that assumption with evidence — the repo's config, symlinks, file layout, existing rules, and official tool or library docs. Cite what you find.
- Explain what makes the change redundant, unnecessary, or worse than the status quo.
- Identify the simplest alternative, and check whether the repo already solves this another way.

If the premise holds after investigation and the change is worth merging, say so plainly instead of manufacturing objections.
