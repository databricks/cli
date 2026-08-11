---
name: bump-sdk
description: "Use when bumping the databricks-sdk-go dependency, upgrading the Go SDK, updating the .codegen/_openapi_sha spec SHA, or moving the CLI to the SDK version used by a given Terraform provider release."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, WebFetch, AskUserQuestion
---

# Bump the Go SDK

The SDK version lives in `go.mod` (`github.com/databricks/databricks-sdk-go`) and the pinned spec SHA lives in `.codegen/_openapi_sha`.
These two move as a pair; everything else in this skill is regenerated from them or is fallout you fix by hand.
Do not hand-edit generated files (`.codegen/cli.json`, `cmd/workspace/*`, `cmd/account/*`, `bundle/schema/jsonschema.json`, `bundle/internal/validation/generated/*`, `bundle/direct/dresources/resources.generated.yml`, `bundle/terraform_dabs_map/generated.go`, `python/databricks/bundles/**`); regenerate them.

The Python tasks (`pydabs-*`, and the `pydabs-codegen` step inside `generate-check`) all run through `uv`. If one fails because `uv` is missing or because the host's `python3` is too old (e.g. 3.9), install `uv` (`curl -LsSf https://astral.sh/uv/install.sh | sh`) rather than touching the system Python: `uv run` provisions the interpreter each package pins (`>=3.10`, and `==3.13.*` under `python/codegen/`) and downloads it if needed. Do not chase the system Python version.

## Steps

**1. Resolve the target versions.**
Use the SDK version the user gave. If they didn't specify one, look up the latest released version yourself (the newest `vX.Y.Z` tag at `https://github.com/databricks/databricks-sdk-go/tags`) instead of prompting for it.
If they want "the version the Terraform provider vX.Y.Z uses", read the provider's `go.mod` at that tag: `https://raw.githubusercontent.com/databricks/terraform-provider-databricks/vX.Y.Z/go.mod`.
The accompanying OpenAPI SHA is pinned inside the SDK at that tag: `https://raw.githubusercontent.com/databricks/databricks-sdk-go/vX.Y.Z/.codegen/_openapi_sha`.
Always bump the spec SHA together with the SDK; they are expected to move as a pair.

**2. Apply the core bump.**
Run `go get github.com/databricks/databricks-sdk-go@vX.Y.Z` to update `go.mod`/`go.sum`.
Write the new SHA with `echo ${SHA} > .codegen/_openapi_sha`; `generate` reformats the file afterward, so a trailing newline here does not matter.
Run `go mod tidy` and confirm the `go.mod`/`go.sum` diff is the SDK line only.

**3. Regenerate cli.json via genkit.**
Run `./task generate-clijson`, which needs a clean universe checkout at `$UNIVERSE_DIR` (default `$HOME/universe`) and builds genkit from its HEAD.
This task authenticates to fetch the spec by SHA, so an SSO/S3 fallback prompt during the run is normal.
The task also syncs `internal/genkit/tagging.py`, its two `.lock` files, and `.github/workflows/tagging.yml` from the producer HEAD; this is drift unrelated to the SDK bump.
Decide with the user whether to keep the tagging-producer drift or restore those files to `origin/main`.

**4. Regenerate everything downstream.**
Run `./task generate-cligen` to regenerate the command stubs from the refreshed `.codegen/cli.json`.
Run `./task generate-check` to regenerate the reproducible artifacts (schema, validation, direct-engine YAML, refschema, pydabs); a clean tree afterwards means zero drift.
Regenerate the DABs<->TF field map with `./task generate-schema-map`, which `generate-check` does not run.
Run `go build ./...` and fix compile breakages before touching acceptance goldens.

**5. Handle SDK breaking changes.**
Read the SDK's `CHANGELOG.md` at the target version (in the module cache) to enumerate breaking changes before chasing compile errors.
A removed struct field that the CLI used (e.g. `jobs.AiRuntimeTask.CodeSourcePath`) should have its usage temporarily disabled with a comment noting it returns in a later SDK bump, not deleted outright.
A new struct field triggers an `exhaustruct` lint failure in `bundle/direct/dresources/*`. Run `./task lint` and (in case of issues) wire the field through `PrepareState` and `RemapState` when it exists on both the input and remote types.
A field the new spec now annotates as output-only may already be emitted into `resources.generated.yml`, making the manual entry in `resources.yml` redundant; `TestResourcesYMLNoRedundantRules` catches this, so remove the manual entry.

**6. Refresh goldens, then VERIFY.**

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

**7. Handle acceptance-test fallout** if the verify pass fails.
A field that becomes required in the spec makes the CLI emit a client-side "required field ... is not set" warning wherever a test config omits it.
For a test that legitimately sets the field, add a non-empty value; if a sibling Terraform-provider bump PR is open, reuse its exact value to avoid conflicts.
For a test that deliberately omits the field to exercise a backend error, that test is likely now redundant with generic client-side validation, so confirm with the user and `git rm` it.
Regenerate the affected test's `out*` files with `go test ./acceptance -run 'TestAccept/<path>' -update` (never hand-edit them), then re-run the full verify pass.

**8. Verify the rest.**
Run `./task fmt` and `./task lint-q`; if either touches `acceptance/`, a fixture is wrong, so fix the source rather than editing output.
Run the full root-module unit suite with `./task test-unit-root`.
If a test fails, confirm whether it is pre-existing by reproducing it on the base commit in a throwaway worktree (`git worktree add --detach /tmp/base HEAD`) before assuming the bump caused it.
Confirm no internal proxy URL leaked into any lock file: `./task check-uv-lock` covers `*uv.lock`, but the genkit `*.py.lock` files are outside that glob, so grep them for `pypi-proxy.cloud.databricks.com` separately.
The `check-uv-lock` glob and the genkit lock revert are a known coverage gap; the internal proxy can re-leak into `internal/genkit/*.py.lock` on any future `generate-clijson`, so re-check after every run.

**9. Changelog fragment.**
Add a `dependency-updates` entry per the `pr-checklist` skill's "Changelog entry" section, modeled on prior bumps: ``Bump `github.com/databricks/databricks-sdk-go` from vOLD to vNEW.``.
Never reference the Terraform provider version in the changelog fragment or PR body.
Add it without `(#NNNN)` now; backfill the number after the PR exists, then run `./task links` to expand it into the full markdown link in place and commit the result.

**10. Commit, push, PR.**
If the push 403s, the active gh account lacks write access to `databricks/cli`; switch to one that has it with `gh auth switch`.
Then follow the `pr-checklist` skill for the commit body, PR description, and changelog, and do not run `gh pr create` without the user's explicit permission.
The bump-specific content for that template is a `## Changes` line ``Bump `github.com/databricks/databricks-sdk-go` from v{old_version} to v{version}.`` followed by one terse bullet per user-visible breaking change or new field handled.

## Stacking and rebasing

If a Terraform-provider bump PR is open and touches the same files (e.g. a value in `bundle/terraform_dabs_map/generated.go`), stack on it or merge it once landed.
On any merge/rebase conflict in a generated file like `generated.go`, resolve to a compilable state and then regenerate rather than hand-merging.
A squash-merged upstream PR leaves its original commits as non-ancestors, so the branch log may show duplicate commits even when the net diff vs `origin/main` is correct.
