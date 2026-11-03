---
name: bump-sdk
description: "Use when bumping the databricks-sdk-go dependency, finishing an automated \"Update Go SDK to vX.Y.Z\" PR from the SDK bumper, upgrading the Go SDK, updating the .codegen/_openapi_sha spec SHA, or moving the CLI to the SDK version used by a given Terraform provider release."
user-invocable: true
allowed-tools: Read, Edit, Write, Bash, Glob, Grep, WebFetch, AskUserQuestion
---

# Bump the Go SDK

The SDK version lives in `go.mod` (`github.com/databricks/databricks-sdk-go`) and the pinned spec SHA lives in `.codegen/_openapi_sha`.
These two move as a pair; everything else in this skill is regenerated from them or is fallout you fix by hand.
Do not hand-edit generated files (`.codegen/cli.json`, `cmd/workspace/*`, `cmd/account/*`, `bundle/schema/jsonschema.json`, `bundle/internal/validation/generated/*`, `bundle/direct/dresources/configs/*.generated.yml`, `python/databricks/bundles/**`); regenerate them.

The Python tasks (`pydabs-*`, and the `pydabs-codegen` step inside `generate-check`) all run through `uv`. If one fails because `uv` is missing or because the host's `python3` is too old (e.g. 3.9), install `uv` (`curl -LsSf https://astral.sh/uv/install.sh | sh`) rather than touching the system Python: `uv run` provisions the interpreter each package pins (`>=3.10`, and `==3.13.*` under `python/codegen/`) and downloads it if needed. Do not chase the system Python version.
The acceptance goldens are the exception: they need CI's `uv` index, see step 4.

## The SDK bumper

The SDK bumper (eng-dev-ecosystem automation) does steps 1-4 for every new stable Go SDK release.
It opens a PR from the `automation/go-sdk-vX.Y.Z` branch of `databricks/cli`, titled `Update Go SDK to vX.Y.Z`, containing the `go get` + `go mod tidy` result, `.codegen/_openapi_sha`, `.codegen/cli.json` (genkit, followed by `./task generate-clijson-post`), and a changelog fragment without a PR link.
`generate-clijson-post` ends with `bump-sdk-post` (step 4), whose error it ignores so that the bumper, which aborts on any error there, still opens a PR. The PR therefore carries whatever got regenerated, and stays red while the bump needs hand fixes (step 5).
Nothing logs the task's output in the bumper, so judge from the PR what got regenerated.
The bumper's environment has `uv` and Python but no `ruff`, which the acceptance tests require, so `test-update` stops at `TestAccept/prerequisites` there and bumper PRs carry no golden updates.
Bumper PRs opened before `generate-clijson-post` ran `bump-sdk-post` (e.g. #7138) have nothing downstream of `cli.json`: their stale command stubs usually fail to compile, which fails lint and every test job.
The bumper does not merge CLI PRs, and it reuses an open PR for the same release on later runs, keeping commits pushed to its branch.

So for a release the bumper already picked up, finish its PR instead of starting a new bump:

- Find it with `gh pr list --repo databricks/cli --head automation/go-sdk-vX.Y.Z`.
- Check its branch out in a worktree (it lives on `databricks/cli`, not a fork) and `git merge origin/main` into it: CI tests the PR merged into `main`, so the goldens must be produced against the same base.
- Continue at step 4: rerun `./task bump-sdk-post` against the merged base and in your environment, which also shows what it failed on in the bumper.

## Steps

Steps 1-4 are the manual equivalent of the bumper (no bumper PR yet, a pinned version, or the version a given Terraform provider uses).

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
Run `./task generate-clijson`, which clones `$UNIVERSE_DIR` (default `$HOME/universe`) into a temporary directory and builds genkit and the OpenAPI spec at `.codegen/_openapi_sha`. If that commit is absent locally, the task fetches `master` from the Universe checkout's `origin` remote into the temporary clone.
The task also syncs `internal/genkit/tagging.py`, its two `.lock` files, and `.github/workflows/tagging.yml` from that Universe commit; this is drift unrelated to the SDK bump.
Decide with the user whether to keep the tagging-producer drift or restore those files to `origin/main`.
The task ends with step 4 (`generate-clijson-post` runs `bump-sdk-post`) and ignores its error, so read the end of its output, or go on with step 4 directly.

**4. Regenerate everything downstream: `./task bump-sdk-post`.**
This is the whole mechanical part of the bump: `generate-check` (command stubs, refschema, schema, validation, direct-engine YAML, pydabs), then `test-update` for the acceptance goldens.
Everything after it needs judgment.
Run directly, it fails on errors: `generate-refschema` and `test-update` build the CLI, so it fails while the bump still breaks compilation; fix that (step 5) and rerun the task.

Run it in CI's environment, or the goldens pick up local drift:
- `uv` index: the acceptance tests build wheels offline from the vendored packages (`UV_OFFLINE`, `UV_FIND_LINKS`), but `uv` still reads its default index's pages from the shared cache. If a `uv.toml` (e.g. `/etc/uv/uv.toml`, which applies even under the tests' temporary `HOME`) makes an internal PyPI proxy the default index, `uv` picks versions from cached proxy pages that it cannot download offline and fails with `No solution found when resolving: hatchling`; `-update` then bakes that error into dozens of template goldens and deletes their `output/` trees. The same failure on `main` tells it apart from bump fallout. Set `UV_DEFAULT_INDEX=https://pypi.org/simple` for the acceptance runs only, so `uv` stops reading cached proxy pages (CI's test jobs point `UV_INDEX_URL` at a JFrog mirror instead): the generators fetch tools such as `ruff` and need the proxy where `pypi.org` is unreachable. On such a machine, run the task's two halves separately: `./task generate-check`, then `UV_DEFAULT_INDEX=https://pypi.org/simple ./task test-update` (and the same prefix for `./task test` later). If `generate-clijson` already ran `test-update` without it, first revert the `acceptance/` churn (`git checkout -- acceptance`, plus any stray untracked files it left).
- Python: CI installs the managed interpreters with `./task install-pythons`; do the same.
- Go: the `toolchain` in `go.mod`. A newer local `go` also changes goldens and generated `gofmt` output; use `GOTOOLCHAIN=<pinned>` if they differ.

Before going on, account for every changed file outside the generated set, including the bumper's (its environment is not CI's either): a bump without fallout changes `cmd/**`, `bundle/schema/jsonschema.json`, `bundle/internal/validation/generated/*`, `python/databricks/bundles/**`, `acceptance/bundle/refschema/output` and help-text goldens (`acceptance/help/output.txt`, `acceptance/cmd/account/account-help/output.txt`).
Revert any golden change you can't attribute to the new SDK or spec.

**5. Handle SDK breaking changes.**
Read the SDK's `CHANGELOG.md` at the target version (in the module cache) to enumerate breaking changes before chasing compile errors.
A removed struct field that the CLI used (e.g. `jobs.AiRuntimeTask.CodeSourcePath`) should have its usage temporarily disabled with a comment noting it returns in a later SDK bump, not deleted outright.
A new struct field triggers an `exhaustruct` lint failure in `bundle/direct/dresources/*`. Run `./task lint` and (in case of issues) wire the field through `PrepareState` and `RemapState` when it exists on both the input and remote types.
Where a remote is rebuilt by hand from a GET response (e.g. `makeJobRunRemote` maps `jobs.Run` into the `jobs.RunNow`-shaped `JobRunRemote`), copy the new field from the response if the response has it, and list it with the request-only zero values otherwise.
`job_runs.yml` must list every `jobs.RunNow` field under `ignore_remote_changes`; `TestJobRunIgnoresEveryRequestField` fails on a new one.
A field the new spec now annotates as output-only may already be emitted into `<resource_type>.generated.yml`, making the manual entry in `<resource_type>.yml` redundant; `TestResourcesYMLNoRedundantRules` catches this, so remove the manual entry.
Rerun `./task bump-sdk-post` after the fixes.

**6. VERIFY.**

```bash
go test ./acceptance -run '^TestAccept$' -timeout=60m       # MUST pass on its own
```

The verify pass is not optional.
Many bundle tests run under an `EnvMatrix` (e.g. `DMS`, `READPLAN`).
When a schema change makes one variant error while another succeeds, the variants produce different output; `-update` runs all of them and each overwrites the other's `output.txt`, so it can silently settle on the passing variant and report `ok` while the golden is actually wrong.
Only the non-update run catches this.
(Ignore `rejecting_proxy.go: blocking proxy` log lines, which are normal.
A test that times out under full parallel load but passes when run alone is a flake, not a regression.)

**7. Handle acceptance-test fallout** if the verify pass fails.
A field that becomes required in the spec makes the CLI emit a client-side "required field ... is not set" warning wherever a test config omits it.
For a test that legitimately sets the field, add a non-empty value; if a sibling Terraform-provider bump PR is open, reuse its exact value to avoid conflicts.
For a test that deliberately omits the field to exercise a backend error, that test is likely now redundant with generic client-side validation, so confirm with the user and `git rm` it.
Regenerate the affected test's `out*` files with `go test ./acceptance -run 'TestAccept/<path>' -update` (never hand-edit them), then re-run the full verify pass.

**8. Verify the rest.**
Run the full `pr-checklist` (`./task fmt`, `./task checks`, `./task lint`, `./task test`, and the pydabs suite, since `python/` changes); if `fmt` or `lint` touches `acceptance/`, a fixture is wrong, so fix the source rather than editing output.
`./task checks` includes `check-lockfiles`, which fails if the internal PyPI proxy URL leaked into a `*uv.lock` or genkit `*.py.lock`.
If a test fails, confirm whether it is pre-existing by reproducing it on the base commit in a throwaway worktree (`git worktree add --detach /tmp/base HEAD`) before assuming the bump caused it.

**9. Changelog fragment.**
The bumper writes `.nextchanges/dependency-updates/go-sdk-vX.Y.Z.md` as `* Update the Go SDK to [vX.Y.Z](https://github.com/databricks/databricks-sdk-go/releases/tag/vX.Y.Z).`; for a manual bump, write the same file so both read alike.
Never reference the Terraform provider version in the changelog fragment or PR body.
The trailing PR link is missing until the PR exists, and the `Changelog Preview` check fails until it is there: append `([#NNNN](https://github.com/databricks/cli/pull/NNNN))` after the period and commit the result.

**10. Commit, push, PR.**
If the push 403s, the active gh account lacks write access to `databricks/cli`; switch to one that has it with `gh auth switch`.

For a bumper PR, commit the fixes on top of the bumper's commit and push to its branch (`git push origin HEAD:automation/go-sdk-vX.Y.Z`); do not rewrite the bumper's commit.
The commit body lists the hand fixes, one terse bullet each.

For a manual bump, follow the `pr-checklist` skill for the commit body, PR description, and changelog, and do not run `gh pr create` without the user's explicit permission.
The bump-specific content for that template is a `## Changes` line ``Bump `github.com/databricks/databricks-sdk-go` from v{old_version} to v{version}.`` followed by one terse bullet per user-visible breaking change or new field handled.

## Stacking and rebasing

On any merge/rebase conflict in a generated file like `generated.go`, resolve to a compilable state and then regenerate rather than hand-merging.
A squash-merged upstream PR leaves its original commits as non-ancestors, so the branch log may show duplicate commits even when the net diff vs `origin/main` is correct.
