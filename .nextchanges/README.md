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
- A PR link is optional. If you want one, write `(#5464)` and run `task links`
  (or `task checks`) to expand it into a full markdown link in place; CI fails
  if a raw `(#5464)` is left unexpanded. The release does not expand links, so
  the fragment must already be expanded when it lands.
- One file is usually one entry; for several, put each on its own `* ` line.

### Sections

| Folder | Section in the released changelog |
| --- | --- |
| `.nextchanges/notable-changes/` | Notable Changes (prominent, called out at the top) |
| `.nextchanges/cli/` | CLI |
| `.nextchanges/bundles/` | Bundles |
| `.nextchanges/dependency-updates/` | Dependency updates |
| `.nextchanges/api-changes/` | API Changes |

See [`.agents/skills/pr-checklist/SKILL.md`](../.agents/skills/pr-checklist/SKILL.md)
for when an entry is warranted.

## How it's released

You don't run anything. At release time the tagging workflow renders every
fragment into the matching section of `CHANGELOG.md`, deletes the consumed
fragments, and bumps `version` to the next minor (see
`internal/genkit/tagging.py`). `./task check-changelog` validates fragment
placement and the `version` file on every PR.

### `version`

`.nextchanges/version` holds the version of the next release (e.g. `1.4.0`).
It's bumped to the next minor automatically after each release — edit it in a
PR only to cut a patch or major release instead.
