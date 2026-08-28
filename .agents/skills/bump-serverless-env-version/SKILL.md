---
name: bump-serverless-env-version
description: "Use when bumping or upgrading the default serverless environment version shipped by Databricks bundle templates, including its coupled Python and DB Connect pins."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, WebFetch, AskUserQuestion
---

# Bump the serverless environment version

Read `.agents/rules/template-versions.md` first for compatibility and
synchronization policy. Never hand-edit generated acceptance output.

## 1. Resolve a compatible version set

Use the requested environment version, or the newest published version when none
was specified. Confirm its runtime Python version from the
[environment release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/)
and [databricks/environments](https://github.com/databricks/environments).

Apply `.agents/rules/template-versions.md` to the environment, Python, and DB
Connect pins. Check Python compatibility in the
[DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements).
Advance `conservative_db_connect_version_spec` only when its DBR line is no longer
supported, using the supported-LTS table in the
[Databricks Runtime release notes](https://docs.databricks.com/aws/en/release-notes/runtime/).
Do not advance it merely to match the environment version.

## 2. Update every template source

Find both macro definitions and hardcoded literals; trust the search results:

```bash
grep -rn 'environment_version\|environment-version' libs/template/templates/
```

Inspect every hit and change only values that pin a version. Re-run the command
after editing and confirm every version-bearing template hit uses the target.
If all sources and coupled pins already match the resolved set and the worktree has
no partial bump, report that no changes are needed and stop.

Current sources include:

- `serverless_environment_version` definitions in `default/library/versions.tmpl`
  and `dbt-sql/library/versions.tmpl`;
- a hardcoded value in the `default-scala` job template;
- three hardcoded values in `lakeflow-integrations`, including its
  `--environment-version` argument.

In `default/library/versions.tmpl`, update `python_version_spec` and
`default_python_version` when the runtime Python version changes. Update
`conservative_db_connect_version_spec` only under the support rule above. Keep the
version-specific compatibility comments accurate even when their pin is unchanged.
Do not synchronize unrelated DBR or SQL-template DB Connect macros.

Update version-specific examples in `.agents/rules/template-versions.md` so its
policy remains accurate; do not change the policy itself as part of the bump.

Also inspect `defaultServerlessVersion` in `libs/localenv/envkey.go`. It is the
product-spec fallback for `databricks environments setup-local`, not a template
pin; do not infer that it should move with the templates. Change it only when the
user or current product specification also requires the fallback to move, and first
confirm `python/serverless/serverless-v{N}/pyproject.toml` exists in
`databricks/environments`. Then update default-version help, error, and test
expectations under `libs/localenv` and `cmd/environments`; verify with
`go test ./libs/localenv ./cmd/environments`, then update and verify with:

```bash
go test ./acceptance -run '^TestAccept/localenv' -update -timeout=60m
go test ./acceptance -run '^TestAccept/localenv' -timeout=60m
```

Otherwise leave it unchanged and record why. Do not change the intentionally older
SSH fixtures in `acceptance/ssh/connect-serverless-*`.

## 3. Regenerate and verify targeted goldens

Update and verify both template acceptance trees:

```bash
./task test-update-templates
go test ./acceptance -run '^TestAccept/pipelines' -update -timeout=60m

go test ./acceptance -run '^TestAccept/bundle/templates' -timeout=60m
go test ./acceptance -run '^TestAccept/pipelines' -timeout=60m
```

Both non-update commands must pass. Update mode selects covering `EnvMatrix`
variants; the non-update runs verify every variant against the regenerated goldens.

## 4. Add the changelog fragment

Add `.nextchanges/bundles/serverless-environment-version-v{N}.md`. Follow the
`pr-checklist` skill's changelog conventions. Describe a benefit stated in the
target version's release notes, link the actual version page, and mention the DB
Connect version only if it changed.

Cross-check the final source and generated-output footprint against the prior
template bumps in [PR #3897](https://github.com/databricks/cli/pull/3897) and
[PR #6378](https://github.com/databricks/cli/pull/6378). Explain material
differences in the final handoff or PR description. Prior PRs are not sources of
truth: revalidate their compatibility decisions, wording, URLs, and file lists.

## 5. Finish only when requested

**Required sub-skill:** use `pr-checklist`, run its checks, and inspect the final
diff. If formatting or linting changes generated acceptance files, fix the source
and regenerate them.

Commit, push, or create/update a PR only when the user explicitly requests that
operation. When requested, follow `pr-checklist` rather than duplicating its commit
and PR-body instructions here.
