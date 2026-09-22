---
name: onboard-team-area
description: "Use when onboarding a team, reserving an experimental area, or assigning code ownership in the Databricks CLI through native GitHub CODEOWNERS."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, AskUserQuestion
---

# Onboard a team / area into the CLI

How ownership works here: `.github/CODEOWNERS` uses GitHub's native code-owner reviews. The last matching rule wins, and any owner on that line can approve. Include `@databricks/eng-deco-cli` on every rule so maintainers can approve changes across all areas. Team membership is maintained in GitHub.

Enforcement requires "Require review from Code Owners" in the target branch's ruleset. Adding a CODEOWNERS file alone requests reviews but does not require them.

## Inputs (ask if missing)

- GitHub team slug, e.g. `eng-ai-custom-training`.
- Area/dir name, e.g. `air`.
- Experimental or stable? This decides where code lands (see the last section).

## Step 1 — Identify the native GitHub team

Use the team as `@databricks/<team>`. GitHub requires the team to be visible and have explicit write access to the repository for its approvals to count.

Use the team slug supplied by the user when they are arranging team creation separately. Verify its access before enabling enforcement.

## Step 2 — Reserve the directories

Create empty `.gitkeep` placeholders so the owned paths exist before any code lands:

```
experimental/<area>/.gitkeep
acceptance/experimental/<area>/.gitkeep
```

## Step 3 — Map the paths in `.github/CODEOWNERS`

Add rules under an `# <Area>` comment. Specific rules go after the `*` maintainer catch-all and must repeat the maintainer team:

```
/experimental/<area>/             @databricks/eng-deco-cli @databricks/<team>
/acceptance/experimental/<area>/  @databricks/eng-deco-cli @databricks/<team>
```

## Step 4 — Validate and open the PR

```bash
git diff --check
# Repo quick checks (no Go/Python/YAML changed, so the formatters have nothing to do)
./task checks
```

Check that the new paths exist and every new rule includes the maintainer team. Once the branch is pushed, inspect GitHub's CODEOWNERS diagnostics for invalid entries or team permissions.

No `.nextchanges/` entry; this is ownership/config only. Write the PR using the `.github/PULL_REQUEST_TEMPLATE.md` sections (Why / Changes / Tests).

## Experimental vs stable, and graduation

- **Experimental** — code under `experimental/<area>/`, tests under `acceptance/experimental/<area>/`. Register it under the hidden parent in `cmd/experimental/experimental.go`, or top-level in `cmd/cmd.go` with `Hidden: true` (as `ssh` does). Experimental commands still ship enabled in every release; `Hidden` only removes them from `--help`, it does not gate or compile them out. No `.nextchanges/` entries while experimental. To hand a build to testers, push a `bugbash-<topic>` branch (auto-builds a snapshot) and share the `internal/bugbash/exec.sh` one-liner.
- **Graduating to stable** — `git mv` the feature-complete commands to `cmd/<area>/` + `libs/<area>/`, register them top-level in `cmd/cmd.go`, keep the old `experimental` paths as deprecated cobra aliases (`sub.Hidden = true`, `sub.Deprecated = '...'`), add CODEOWNERS rules for the new stable paths, and add a `.nextchanges/` entry. See `experimental/aitools` graduating to top-level `aitools` (PR #4917) as the worked example.
