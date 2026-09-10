# Changelog fragments

Add a changelog entry by creating a **new file** in the section folder under
`.nextchanges/` that fits your change. Each PR adds its own file, so two PRs
never touch the same path — no merge conflicts, unlike everyone editing one
shared changelog file.

## How to add an entry (takes 10 seconds)

Create `.nextchanges/<section>/<name>.md` and write what changed:

```
* Add the `databricks quickstart` command.
```

You can do this straight from the GitHub UI: **Add file → Create new file**,
type the path (e.g. `.nextchanges/cli/quickstart.md`), write the entry, commit.

- `<name>` is arbitrary — a feature name (`quickstart.md`) or your PR number
  (`5464.md`), whatever you like, as long as it's unique.
- One file is exactly one entry: a single line that starts with a `* ` bullet
  marker and ends with a period. `task check-changelog` (and CI) enforces this.
- A trailing PR link is required whenever the change is associated with a PR,
  and the PR that introduces the entry must be among the linked ones — the
  checker infers that PR (from the squash-merge commit that added the fragment,
  or your open branch PR) and fails if it isn't listed. CI enforces this on
  every PR and on `main`, and `task check-changelog` enforces it locally too
  once your branch has an open PR (detected best-effort via `gh`; skipped before
  the PR exists or when `gh` is unavailable). Write the full markdown link at the
  very end, after the period:
  `([#5464](https://github.com/databricks/cli/pull/5464))` (your PR number). For
  an entry spanning several PRs, list them comma-separated:
  `([#5464](…), [#5500](…))`, as long as the introducing PR is included.
- Every `#5464` reference — inline or trailing — must be a full markdown link.
  A bare or paren-wrapped `#5464` is rejected: GitHub would render it as an
  unintended auto-link in `CHANGELOG.md`. Nothing rewrites links, so the
  fragment must already be correct when it lands.

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
