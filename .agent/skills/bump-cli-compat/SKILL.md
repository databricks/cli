---
name: bump-cli-compat
description: "Bump cli-compat.json with new AppKit and Agent Skills versions, then create a PR. Use when the user says 'bump cli-compat', 'update cli-compat', 'bump compatibility manifest', 'new appkit release cli-compat', or wants to update the CLI compatibility manifest after an AppKit or Agent Skills release."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, AskUserQuestion
---

# Bump CLI Compatibility Manifest

Updates `internal/build/cli-compat.json` with new AppKit and Agent Skills versions, validates the result, and creates a PR.

## Arguments

Parse the user's input for optional version arguments:

- `--appkit <version>` or first positional arg → AppKit version (e.g. `0.28.0`)
- `--skills <version>` or second positional arg → Agent Skills version (e.g. `0.1.6`)
- No args → auto-detect latest versions from GitHub tags

Versions should be provided **without** the `v` prefix (e.g. `0.28.0`, not `v0.28.0`). If provided with the prefix, strip it.

## Workflow

### Step 1: Resolve versions

If both `appkit` and `skills` versions were provided as arguments, skip to Step 2.

Otherwise, fetch the latest tags from GitHub:

```bash
# Latest appkit version (strip leading 'v')
gh api repos/databricks/appkit/tags --jq '.[0].name' | sed 's/^v//'

# Latest skills version (strip leading 'v')
gh api repos/databricks/databricks-agent-skills/tags --jq '.[0].name' | sed 's/^v//'
```

Show the resolved versions to the user and ask:

> The latest versions are:
> - AppKit: `{appkit_version}`
> - Agent Skills: `{skills_version}`
>
> Have these versions been evaluated (evals passed with no regressions)?

**Do NOT proceed until the user confirms.** If the user says no or wants different versions, ask them to provide the correct versions.

### Step 2: Validate tags exist

Verify that the corresponding Git tags exist on GitHub:

```bash
gh api repos/databricks/appkit/git/ref/tags/v{appkit_version} --jq '.ref' 2>&1
gh api repos/databricks/databricks-agent-skills/git/ref/tags/v{skills_version} --jq '.ref' 2>&1
```

If either tag doesn't exist, report the error and stop.

### Step 3: Read current manifest

Read `internal/build/cli-compat.json`. Note the current versions and the list of versioned entries.

### Step 4: Update the manifest

Update **all entries** (both `next` and all versioned CLI entries) to the new appkit and skills versions. This is the "no template changes" scenario — a simple search & replace.

Write the updated `internal/build/cli-compat.json`.

### Step 5: Validate

Run the Go tests to ensure the manifest is well-formed:

```bash
go test ./libs/depversions/... -run TestEmbeddedManifest -v
```

If validation fails, show the errors and fix them before proceeding.

### Step 6: Create branch, commit, and PR

```bash
# Create a new branch from the current branch (or main)
git checkout -b bump-cli-compat-appkit-{appkit_version}-skills-{skills_version}

# Stage and commit
git add internal/build/cli-compat.json
git commit -s -m "chore: bump cli-compat to appkit {appkit_version}, skills {skills_version}"

# Push and create PR
git push -u origin HEAD
gh pr create \
  --title "chore: bump cli-compat to appkit {appkit_version}, skills {skills_version}" \
  --body "$(cat <<'EOF'
## Summary
Bump `cli-compat.json` to use:
- AppKit `{appkit_version}`
- Agent Skills `{skills_version}`

## Checklist
- [ ] Evals passed with no regressions
- [ ] `go test ./libs/depversions/... -run TestEmbeddedManifest` passes
EOF
)"
```

Show the PR URL to the user when done.

## Examples

### Example: With explicit versions
```
/bump-cli-compat 0.28.0 0.1.6
```
Validates tags exist, updates manifest, creates PR.

### Example: Auto-detect latest
```
/bump-cli-compat
```
Fetches latest tags, asks for eval confirmation, then updates and creates PR.

### Example: With flags
```
/bump-cli-compat --appkit 0.28.0 --skills 0.1.6
```
Same as positional args.
