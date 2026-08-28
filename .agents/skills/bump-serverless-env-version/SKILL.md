---
name: bump-serverless-env-version
description: "Use when bumping the default serverless environment version in bundle templates, upgrading serverless env to a new version, or updating the coupled DB Connect / Python version pins in libs/template/templates/*/library/versions.tmpl."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, WebFetch, AskUserQuestion
---

# Bump the serverless environment version

The DB Connect and Python pins a freshly initialized bundle project ships with
live in the `library/versions.tmpl` files under `libs/template/templates/`. The
serverless environment version lives in two forms — a `versions.tmpl` macro *and*
hardcoded literals in a few templates that don't reference the macro (Step 2).
`serverless_environment_version`, the Python pins, and the DB Connect pin form
one mutually compatible set; everything under `acceptance/` is rendered output
you regenerate, not hand-edit.

**Read `.agents/rules/template-versions.md` first** — it holds the three RULES this
skill must obey (keep DB Connect at the lowest working pin, keep the version set
mutually compatible, sync `serverless_environment_version` across templates but
nothing else). This skill is the procedure; that doc is the policy.

**Scope:** bundle template pins. Two related version pins live outside the
templates — check them, don't assume:
- `defaultServerlessVersion` in `libs/localenv/envkey.go` is a separate Go pin: the
  fallback serverless version for `databricks environments setup-local`, documented
  as a "stand-in for the latest LTS". It currently tracks the template version. If
  the version you're moving to is the latest LTS, bump this constant in the same PR
  and regenerate its tests (`libs/localenv`, `cmd/environments`); if it is not yet
  LTS, leave it and say why. Do not silently ignore it.
- The SSH acceptance goldens (`acceptance/ssh/connect-serverless-*/output.txt`) pin
  an explicit older version on purpose — a deliberate test fixture, not the
  template default. Leave them alone.

## Steps

**1. Resolve the target version and cross-check compatibility.**
Use the environment version the user gave, or the newest one on the
[serverless environment version release notes](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/).
An environment version dictates a runtime Python version, and the DB Connect pin
must support that Python, so resolve all three together:
- Confirm the target version exists and note the Python it ships against the
  [databricks/environments](https://github.com/databricks/environments) repo (public — the
  source of truth for each environment version's package set) and the release notes.
- Check the DB Connect pin against the
  [DB Connect requirements](https://docs.databricks.com/dev-tools/databricks-connect/python/index.html#requirements).
- Apply the RULES in `.agents/rules/template-versions.md`. In particular, bump
  `conservative_db_connect_version_spec` **only** when the currently pinned DBR
  release has fallen out of support — not merely to match the new environment
  version.

**2. Apply the edits.** The environment version lives in two forms — a macro and
hardcoded literals — so this grep, not any file list, is the source of truth for
what to change. Bumping the macro alone silently misses the literals:

```bash
grep -rn 'environment_version\|environment-version' libs/template/templates/
```

Bring **every** hit to the target version, then re-run the grep and confirm each
one shows the new value. The hits fall into two kinds (paths below are what they
match today, for orientation — trust the grep if they've moved):
- **Macro** (`serverless_environment_version`): defined in
  `default/library/versions.tmpl` and `dbt-sql/library/versions.tmpl` and
  referenced by those templates' job/notebook files. Editing the two `define`s
  updates every reference.
- **Hardcoded literals**: templates that pin the version directly, so the macro
  edit does *not* reach them — currently `default-scala`'s job template and three
  `lakeflow-integrations` files (including a `--environment-version` CLI arg). Edit
  each literal by hand.
- In `default/` also bump `python_version_spec` / `default_python_version` if the
  new environment version's Python changed, and
  `conservative_db_connect_version_spec` only per the rule above.
- Do **not** sync other macros. `latest_lts_dbr_version` is intentionally `16.4`
  in `default/` but `15.4` in the SQL templates, and each SQL template's
  `latest_lts_db_connect_version_spec` is a distinct macro from `default/`'s
  `conservative_db_connect_version_spec`. `default-sql/` does not ship a serverless
  environment version at all.

**3. Update the version comments.**
Each macro in `default/versions.tmpl` carries a comment block explaining the
compatibility reasoning (which Python the environment version uses, why DB
Connect is pinned where it is). Update those so the "why" matches the new pins.

**4. Regenerate goldens across BOTH template trees, then VERIFY.**
Template output is rendered into two acceptance trees, and a bump changes both:
- `acceptance/bundle/templates/**` (default-python, dbt-sql, lakeflow-pipelines, …)
- `acceptance/pipelines/**` (pipelines init renders lakeflow-pipelines)

`./task test-update-templates` regenerates only `bundle/templates` and misses
`pipelines`, so use the full update + verify pass instead:

```bash
GOTOOLCHAIN=local go test ./acceptance -run '^TestAccept$' -update -timeout=60m
GOTOOLCHAIN=local go test ./acceptance -run '^TestAccept$' -timeout=60m   # MUST pass on its own
```

The verify (non-update) pass is not optional. Bundle tests run under an
`EnvMatrix` of both engines (`terraform`, `direct`); with `-update` each variant
overwrites the other's `output.txt`, so a run can report `ok` on a golden that is
actually wrong. Only the non-update run catches this. (Ignore
`rejecting_proxy.go: blocking proxy` log lines — they are normal. A test that
times out under full parallel load but passes when run alone is a flake.)

**5. Changelog fragment.**
Add a `bundles` fragment at `.nextchanges/bundles/serverless-environment-version-v{N}.md`,
modeled on the prior bump:

```
Bundle templates now use serverless [environment version {N}](https://docs.databricks.com/aws/en/release-notes/serverless/environment-version/{word}), which offers better performance, and `databricks-connect` {X.Y}.
```

See the `pr-checklist` skill's "Changelog entry" section for conventions.

**6. Commit, push, PR.**
Run `./task fmt` and `./task lint-q` (if either touches `acceptance/`, a fixture
is wrong — fix the source, not the output). Commit and push; if the push 403s,
the active gh account lacks write access to `databricks/cli` (`gh auth switch`).
Then follow the `pr-checklist` skill for the commit body and PR description.
**Do not run `gh pr create` without the user's explicit permission.**

Commit body and PR description:

```
## Changes

Bump the default serverless environment version in bundle templates from {old} to {N}.

- Python pinned to {python_spec} to match environment version {N}
- DB Connect pinned to {db_connect_spec} <only if changed; state why per the DBR-support rule>

## Tests

Acceptance goldens regenerated across bundle/templates and pipelines.
```
