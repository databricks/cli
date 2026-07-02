---
name: onboard-team-area
description: "Onboard a new team or reserve a new code area in the Databricks CLI: add the team to .github/OWNERTEAMS, reserve experimental/<area>/ and acceptance/experimental/<area>/, and map both paths in .github/OWNERS so the maintainer-approval gate routes the area's PRs to the team. Use when the user says 'onboard a team', 'add an OWNERS team', 'reserve an experimental area', 'add a new team to the CLI', or wants a new owned directory wired into review."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, AskUserQuestion
---

# Onboard a team / area into the CLI

How ownership works here: `.github/OWNERS` is CODEOWNERS-style and last-match-wins. `.github/OWNERTEAMS` maps `team:<name>` to an explicit `@member` list and is the source of truth, because the CI token cannot resolve GitHub org-team membership. The `maintainer-approval` workflow is a required check that blocks merge until every owned group a PR touches has at least one approval from one of its owners.

Worked example: PR #5605 ("Add ai-training OWNERS team and reserve experimental/air") is exactly the steps below: +1 line in OWNERTEAMS, two `.gitkeep` files, +2 lines in OWNERS.

## Inputs (ask if missing)

- Team alias, e.g. `ai-training`, and the `@member` list.
- Area/dir name, e.g. `air`.
- Experimental or stable? This decides where code lands (see the last section).

## Step 1 — Add the team to `.github/OWNERTEAMS`

Append one line, keeping the existing column alignment:

```
team:<name>  @member1 @member2 ...
```

If the team has a GitHub team page, add its URL to the header comment block. Skip the URL if the team page does not exist yet; the validator only warns about a missing URL, it does not block.

## Step 2 — Reserve the directories

Create empty `.gitkeep` placeholders so the owned paths exist before any code lands:

```
experimental/<area>/.gitkeep
acceptance/experimental/<area>/.gitkeep
```

## Step 3 — Map the paths in `.github/OWNERS`

Add rules under an `# <Area>` comment. Because last-match-wins, specific rules go after the `*` maintainer catch-all:

```
/experimental/<area>/             team:<name>
/acceptance/experimental/<area>/  team:<name>
```

## Step 4 — Validate and open the PR

```bash
# OWNERS parser + approval-logic tests
node --test .github/scripts/owners.test.js .github/workflows/maintainer-approval.test.js
# OWNERS/OWNERTEAMS consistency: undefined teams, zero-owner rules, missing paths
node .github/scripts/owners.js validate
# Repo quick checks (no Go/Python/YAML changed, so the formatters have nothing to do)
./task checks
```

No `NEXT_CHANGELOG.md` entry; this is ownership/config only. Write the PR using the `.github/PULL_REQUEST_TEMPLATE.md` sections (Why / Changes / Tests).

## Experimental vs stable, and graduation

- **Experimental** — code under `experimental/<area>/`, tests under `acceptance/experimental/<area>/`. Register it under the hidden parent in `cmd/experimental/experimental.go`, or top-level in `cmd/cmd.go` with `Hidden: true` (as `ssh` does). Experimental commands still ship enabled in every release; `Hidden` only removes them from `--help`, it does not gate or compile them out. No `NEXT_CHANGELOG` entries while experimental. To hand a build to testers, push a `bugbash-<topic>` branch (auto-builds a snapshot) and share the `internal/bugbash/exec.sh` one-liner.
- **Graduating to stable** — `git mv` the feature-complete commands to `cmd/<area>/` + `libs/<area>/`, register them top-level in `cmd/cmd.go`, keep the old `experimental` paths as deprecated cobra aliases (`sub.Hidden = true`, `sub.Deprecated = '...'`), add OWNERS rules for the new stable paths, and add the `NEXT_CHANGELOG` entry. See `experimental/aitools` graduating to top-level `aitools` (PR #4917) as the worked example.
