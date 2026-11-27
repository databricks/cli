This file provides guidance to AI assistants when working with code in this repository. Rules prefixed `**RULE:**` are mandatory. Path-specific rules live in `.agents/rules/`.

# Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Declarative Automation Bundles (DABs), formerly known as Databricks Asset Bundles. The project is written in Go.

# General Rules

**RULE: When moving code from one place to another, don't unnecessarily change or omit parts.** Keep refactors separate from content changes so reviewers can tell them apart.

**RULE: Prefer simplicity over cleverness. Avoid speculative fallbacks and default values.** If you catch yourself adding a fallback branch "just in case," identify the correct path and use only that one.

**RULE: Keep each PR focused on one change.** Put unrelated cleanups, fixes, refactors and dependency bumps in a separate PR.

**RULE: Before adding a new helper, search the codebase for an existing one.** Common homes: `libs/` (notably `libs/cmdio/`, `libs/env/`, `libs/filer/`, `libs/structs/`), `acceptance/bin/` (acceptance test helpers), `internal/mocks/` (generated mocks).

# Development Commands

- `./task build` - Build the CLI binary
- `./task test` - Run unit and acceptance tests for all packages
- `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder> -tail -test.v` - Run a single acceptance test
- `./task test-update` - Regenerate acceptance test outputs
- `./task lint` - Run full linter across all Go modules; `./task lint-q` - changed files only, with `--fix`
- `./task fmt` - Format all code (Go, Python, YAML); `./task fmt-q` - changed files only
- `./task checks` - Run quick checks (tidy, whitespace, links)
- `./task generate` - Run all generators (see `.agents/rules/auto-generated-files.md`)

Every acceptance test runs locally against the fake server in `libs/testserver`. `Cloud = true` in a `test.toml` means the test *also* runs against a real workspace — never that it skips the local run. The `integration/` tree is deprecated; coverage that needs a real workspace goes into an acceptance test with `Cloud = true`. See `.agents/rules/testing.md`.

**RULE: Use `git rm` to remove and `git mv` to rename files.**

**RULE: When rebasing, prefix git commands so they never launch an interactive editor.**

```sh
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git fetch origin main &&
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git rebase origin/main
```

If `./task lint` reports issues in files under a *different* or deleted worktree, the golangci-lint cache is stale. Run `go tool -modfile=tools/go.mod golangci-lint cache clean` and re-run; don't investigate the reported issues.

# Architecture

- `cmd/` - Cobra commands. `cmd/workspace/` and `cmd/account/` are generated.
- `bundle/` - Bundles: `config/` (loading, validation, schema), `config/mutator/` and `mutator/` (the configuration transformation pipeline; each mutator implements `Mutator`), `phases/` (high-level deployment phases), `direct/` (the deployment engine).
- `libs/` - Shared libraries: `filer/` (local, DBFS, workspace file systems), `auth/`, `sync/`.

Bundles are deployed by the direct engine, which calls the APIs without Terraform. The Terraform engine was removed in v1.20.0: `bundle.engine: terraform` and `DATABRICKS_BUNDLE_ENGINE=terraform` are rejected, and existing Terraform state is migrated to the direct engine before the command runs.

The CLI binary supports both `databricks` and `pipelines` command modes based on executable name.

# Common Mistakes

**RULE: Do not remove or skip failing tests to fix CI.** Fix the underlying issue instead.

**RULE: When adding a direct Go dependency, annotate its license in `go.mod` and add it to `NOTICE`.** The allowed SPDX identifiers are the `spdxLicenses` map in `internal/build/license_test.go`. If a dep's license isn't on the allowlist, discuss before adding.

**RULE: Do not use `os.Exit()` in library or command code; `main.go` owns the exit path.** It skips deferred cleanup and complicates testing. `TestMain` does not need it either: since Go 1.15, returning from `TestMain` exits with `m.Run()`'s code.

**RULE: Do not add defensive `nil` checks for values the caller or framework always provides.** For example, direct engine resource methods (`DoCreate`, `DoUpdate`, `RemapState`, etc.) never receive nil receivers or state. Where a panic is genuinely possible (e.g. `reflect.Type.Elem()` on a non-pointer), validate at the entry point and return an error.

**RULE: Use a reserved, non-resolving TLD (`.test`) for any test fixture host** — `Config.Host`, `databricks.yml`'s `workspace.host`, `.databrickscfg`. Real domains hit the SDK well-known endpoint resolver and can stall tests for minutes.

**RULE: Open URLs through `libs/browser`, never `github.com/pkg/browser` directly.** `pkg/browser` ignores the `BROWSER` env var. For a custom `BROWSER=<command>`, launch via `libs/exec`, not an inline `cmd /c` (that broke Windows OAuth URLs via `%...%` expansion).

**RULE: For hand-written workspace-routed raw API calls (`client.Do` / `apiClient.Do`), set routing headers via `libs/auth.WorkspaceIDHeaders`.**

**RULE: To keep an intentionally-unreferenced exported function, mark it `//deadcode:allow <reason>` within 6 lines above the `func`.** CI runs `deadcode -test ./...`.

**RULE: Write platform-agnostic code and tests; they also run on Windows.**

# Errors and CLI UX

**RULE: Never branch on `err.Error()` string content.** Use `errors.Is` / `errors.AsType` with a sentinel or typed error (e.g. `apierr.ErrResourceDoesNotExist`).

**RULE: Use `diag.Errorf` / `diag.Warningf` to create diagnostics and `logdiag.LogDiag` / `logdiag.LogError` to log them.**

**RULE: Reject incompatible inputs early with an actionable error. Never silently ignore a flag or config field the current mode can't honor.** Name the conflicting flags or the field in the error. A config field that applies only to certain resource types is a validation error, not a warning that gets lost in log output.
