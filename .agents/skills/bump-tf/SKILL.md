---
name: bump-tf
description: "Use when the user says 'bump the TF provider', 'bump terraform provider', 'update the terraform provider to <version>', 'bump TF to latest', or otherwise wants the pinned Databricks Terraform provider version raised and the new provider mappings pulled into the CLI."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, AskUserQuestion
---

# Bump Terraform Provider

The pinned provider version lives in `bundle/internal/tf/codegen/schema/version.go`.
Everything else (`bundle/internal/tf/schema/*` including `root.go`, and `bundle/terraform_dabs_map/generated.go`) is generated from it, so the only file you hand-edit for the bump itself is `version.go`.
Do not edit the generated files, and do not touch the `databricks-tf-provider/...` version comment in `libs/testdiff/replacement.go` (the version is masked in acceptance output, so changing it is pure noise).

## Steps

**1. Resolve the target version.**
Use the version the user gave (strip a leading `v`).
For "latest"/no version, resolve the newest GitHub release.
Either way, confirm the tag exists on GitHub, since you can't bump to an unreleased version:

```bash
gh api repos/databricks/terraform-provider-databricks/releases --jq '.[0].tag_name'   # latest
gh api repos/databricks/terraform-provider-databricks/releases/tags/v{version} --jq '.published_at'
```

**2. Bump the constant.**
Set `const ProviderVersion = "{version}"` in `bundle/internal/tf/codegen/schema/version.go`.

**3. Regenerate the schema and field map:**

```bash
./task generate-tf-schema        # regenerates bundle/internal/tf/schema/* (incl. root.go)
gofmt -s -w bundle/internal/tf/schema
./task generate-schema-map       # regenerates bundle/terraform_dabs_map/generated.go
go build ./...
```

A `Warning: Skipping file generation for databricks_quality_monitor ...` line from codegen is expected.
If `generate-tf-schema` fails with `no available releases match the given constraints {version}`, the registry hasn't indexed the release yet.
See [registry-workaround.md](registry-workaround.md), then continue.

**4. Refresh goldens, then VERIFY.**

```bash
go test ./acceptance -run '^TestAccept$' -update -timeout=60m
go test ./acceptance -run '^TestAccept$' -timeout=60m       # MUST pass on its own
```

The verify pass is not optional.
Bundle tests run under an `EnvMatrix` of both engines (`terraform`, `direct`).
When a schema change makes one engine error while the other succeeds, the variants produce different output; `-update` runs both and each overwrites the other's `output.txt`, so it can silently settle on the passing variant and report `ok` while the golden is actually wrong.
Only the non-update run catches this.
(Ignore `rejecting_proxy.go: blocking proxy` log lines, which are normal.
A test that times out under full parallel load but passes when run alone is a flake, not a regression.)

**5. Resolve behavior changes** if the verify pass fails.
Diff the generated schema (`git diff bundle/internal/tf/schema/resource_<x>.go`) and adapt the **test**, not the generated code:

- *Field became required* → the terraform engine rejects a fixture the direct engine still accepts.
  Add the now-required field to the fixture `databricks.yml` (copy the shape from a sibling test) so both engines match again.
- *Field removed* but still supported by the direct engine → terraform warns `unknown field: <field>` and drops it.
  Restrict the test to the direct engine, with a comment:

  ```toml
  # The Terraform provider dropped `<field>` in v{version}; direct engine only.
  EnvMatrix.DATABRICKS_BUNDLE_ENGINE = ["direct"]
  ```

Regenerate the affected test's `out*` files with `go test ./acceptance -run 'TestAccept/<path>' -update` (never hand-edit them), then re-run the full verify pass.

**6. Changelog fragment.**
Add a `dependency-updates` entry per the `pr-checklist` skill's "Changelog entry" section:

```
* Bump Terraform provider from v{old_version} to v{version}. ([#{pr_number}](https://github.com/databricks/cli/pull/{pr_number}))
```

Omit the trailing PR link now (you don't have the number yet); after the PR exists, append `([#NNNN](https://github.com/databricks/cli/pull/NNNN))` after the period and commit the result.

**7. Commit, push, PR.**
Run `./task fmt` and `./task lint-q` (if either touches `acceptance/`, a fixture is wrong, so fix the source rather than editing output).
Commit and push; if the push 403s, the account needs write access to `databricks/cli` (`gh auth switch`).
Then follow the `pr-checklist` skill for the PR.
**Do not run `gh pr create` without the user's explicit permission.**
Commit body and PR description:

```
## Changes

Bump the pinned Databricks Terraform provider from v{old_version} to v{version}.

Notable schema changes:
- <one terse bullet per user-visible resource/field change>

## Tests

Acceptance goldens regenerated via `./task test-update`.
```
