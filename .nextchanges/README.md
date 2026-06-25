# Changelog fragments

Add a changelog entry by creating a **new file** in the section folder under
`.nextchanges/` that fits your change. Each PR adds its own file, so two PRs
never touch the same path — no merge conflicts, unlike everyone editing one
shared changelog file.

## How to add an entry (takes 10 seconds)

Create `.nextchanges/<section>/<name>.md` and write what changed:

```
Added the `databricks quickstart` command.
```

You can do this straight from the GitHub UI: **Add file → Create new file**,
type the path (e.g. `.nextchanges/cli/quickstart.md`), write a sentence, commit.

- `<name>` is arbitrary — a feature name (`quickstart.md`) or your PR number
  (`5464.md`), whatever you like, as long as it's unique.
- The leading `* ` is optional.
- A PR link is optional: write `(#5464)` anywhere in the text and it becomes a
  full link automatically (see `tools/update_github_links.py`).
- One file is usually one entry; for several, put each on its own `* ` line.

### Sections

| Folder | Section in the released changelog |
| --- | --- |
| `.nextchanges/notable-changes/` | Notable Changes (prominent, called out at the top) |
| `.nextchanges/cli/` | CLI |
| `.nextchanges/bundles/` | Bundles |
| `.nextchanges/dependency-updates/` | Dependency updates |
| `.nextchanges/api-changes/` | API Changes |

See [`.agent/skills/pr-checklist/SKILL.md`](../.agent/skills/pr-checklist/SKILL.md)
for when an entry is warranted.

## How it's released

You don't run anything. At release time, `tools/collate_changelog.py` folds
every fragment into the matching section of `NEXT_CHANGELOG.md`, deletes the
fragments, and the release tooling generates `CHANGELOG.md` as before.
`./task changelog-check` validates fragment placement on every PR.
