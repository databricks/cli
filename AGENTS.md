This file provides guidance to AI assistants when working with code in this repository.

Rules prefixed `**RULE:**` are mandatory. `GOOD:` and `BAD:` labels on code snippets mark patterns to follow and patterns to avoid. This convention is a common best practice for AI-assistant rule files and is used consistently across `AGENTS.md` and `.agent/rules/*.md`.

# Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Declarative Automation Bundles (DABs), formerly known as Databricks Asset Bundles. The project is written in Go and follows a modular architecture.

# General Rules

**RULE: When moving code from one place to another, don't unnecessarily change or omit parts.** Keep refactors separate from content changes so reviewers can tell them apart.

**RULE: Do not modify or remove existing comments in code you didn't write.** Comments often encode non-obvious context (a bug reference, a workaround, a reason the code is shaped a certain way) that is lost if rewritten. Leave them alone unless the user explicitly asks for a change.

**RULE: Prefer simplicity over cleverness. Avoid speculative fallbacks and default values.** If you catch yourself adding a fallback branch "just in case," identify the correct path and use only that one. Reviewers in this repo reject speculative flexibility.

**RULE: Keep each PR focused on one change.** If you notice an unrelated cleanup, bug fix, or refactor while making your primary change, leave it alone or put it in a separate PR. Reviewers consistently ask to split mixed PRs, especially when a dependency bump or schema diff rides along with a feature change.

**RULE: Before adding a new helper, search the codebase for an existing one.** Common homes: `libs/` (shared utilities), `libs/databrickscfg/` (config), `libs/git/`, `libs/filer/`, `libs/cmdio/` (CLI I/O, spinners, prompts), `libs/env/` (env vars), `libs/testserver/`, `libs/structpath/` and `libs/dyn/` (path / dynamic values), `acceptance/bin/` (acceptance test helpers), `internal/mocks/` (generated mocks). A function that duplicates an existing name and signature in the same package is a compile error waiting to happen; grep before you name.

# Development Commands

### Building and Testing

- `make build` - Build the CLI binary
- `make test` - Run unit tests for all packages
- `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder> -tail -test.v` - run a single acceptance test
- `make integration` - Run integration tests (requires environment variables)
- `make cover` - Generate test coverage reports

### Code Quality

- `make lint` - Run linter on changed files only (uses lintdiff.py)
- `make lintfull` - Run full linter with fixes (golangci-lint)
- `make ws` - Run whitespace linter
- `make fmt` - Format code (Go, Python, YAML)
- `make checks` - Run quick checks (tidy, whitespace, links)

### Specialized Commands

- `make schema` - Generate bundle JSON schema
- `make docs` - Generate bundle documentation
- `make generate` - Generate CLI code from OpenAPI spec (requires universe repo)

### Git Commands

**RULE: Use `git rm` to remove and `git mv` to rename files, instead of directly modifying files on the filesystem.**

**RULE: When rebasing, prefix git commands so they never launch an interactive editor.**

```sh
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git fetch origin main &&
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git rebase origin/main
```

# Architecture

### Core Components

**cmd/** - CLI command structure using Cobra framework
- `cmd/cmd.go` - Main command setup and subcommand registration
- `cmd/bundle/` - Bundle-related commands (deploy, validate, etc.)
- `cmd/workspace/` - Workspace API commands (auto-generated)
- `cmd/account/` - Account-level API commands (auto-generated)

**bundle/** - Core bundle functionality for Declarative Automation Bundles
- `bundle/bundle.go` - Main Bundle struct and lifecycle management
- `bundle/config/` - Configuration loading, validation, and schema
- `bundle/deploy/` - Deployment logic (Terraform and direct modes)
- `bundle/mutator/` - Configuration transformation pipeline
- `bundle/phases/` - High-level deployment phases

**libs/** - Shared libraries and utilities
- `libs/dyn/` - Dynamic configuration value manipulation
- `libs/filer/` - File system abstraction (local, DBFS, workspace)
- `libs/auth/` - Databricks authentication handling
- `libs/sync/` - File synchronization between local and remote

### Key Concepts

**Bundles**: Configuration-driven deployments of Databricks resources (jobs, pipelines, etc.). The bundle system uses a mutator pattern where each transformation is a separate, testable component.

**Mutators**: Transform bundle configuration through a pipeline. Located in `bundle/config/mutator/` and `bundle/mutator/`. Each mutator implements the `Mutator` interface.

**Direct vs Terraform Deployment**: The CLI supports two deployment modes controlled by `DATABRICKS_BUNDLE_ENGINE` environment variable:
- `terraform` (default) - Uses Terraform for resource management
- `direct` - Direct API calls without Terraform

# Development Tips

- Use `make test-update` to regenerate acceptance test outputs after changes.
- The CLI binary supports both `databricks` and `pipelines` command modes based on executable name.

**RULE: Comments should explain "why", not "what".** Reviewers consistently reject comments that merely restate the code.

**RULE: When code relies on a non-obvious invariant, workaround, or backend quirk, add a short comment stating the reason.** The inverse of the rule above: noise comments are bad, but missing comments are the single most common thing reviewers catch. Triggers include: API quirks (PATCH-like semantics, no get-by-name, stripped prefixes), fields intentionally included or excluded (output-only, etag, `ForceSendFields`), branches that look dead but are kept as guards, and tests where the expectation isn't obvious from the assertions.

GOOD:

```go
// The Workspace API strips the "/Workspace" prefix from parent_path on GET,
// so we re-add it here to match the local configuration.
parentPath = "/Workspace" + parentPath
```

BAD:

```go
parentPath = "/Workspace" + parentPath
```

# Common Mistakes

**RULE: When adding a direct Go dependency, annotate its license in `go.mod` and update `NOTICE`.** Before picking the SPDX identifier, read `internal/build/license_test.go` to see the current allowlist (the `spdxLicenses` map). That test is the source of truth and will fail CI if a direct `require` line lacks a matching SPDX suffix comment (e.g. `// MIT`). Also add a corresponding entry to `NOTICE` under the matching license section. If a dep's license isn't on the allowlist, discuss before adding.

**RULE: Do not use `os.Exit()` outside of `main.go`.** `main.go` owns the exit path; calling `os.Exit()` elsewhere skips deferred cleanup and complicates testing.

**RULE: Do not remove or skip failing tests to fix CI.** Fix the underlying issue instead.

**RULE: Do not leave debug print statements in committed code.** `fmt.Println`, `log.Printf`, or similar. Always scrub before committing.

**RULE: Do not add defensive `nil` checks for values the caller or framework is documented to always provide.** If a check exists "just in case", either remove it or attach a comment explaining why the invariant might be violated. Direct engine resource methods (`DoCreate`, `DoUpdate`, `RemapState`, etc.) never receive nil receivers or state from the framework, so extra nil-guards there are dead code.

Where a panic is genuinely possible (e.g. `reflect.Type.Elem()` on a non-pointer, division by an empty slice's length), validate at the entry point and return an error.

# Error Handling

**RULE: Wrap errors with context using `%w`.** Preserves the error chain so `errors.Is` and `errors.As` keep working upstream.

GOOD:

```go
return fmt.Errorf("failed to deploy %s: %w", name, err)
```

BAD:

```go
return fmt.Errorf("failed to deploy %s: %s", name, err)
```

**RULE: Return early on errors; avoid deeply nested if-else chains.**

**RULE: Use `logdiag.LogDiag` and `logdiag.LogError` for logging diagnostics.**

**RULE: Use `diag.Errorf` and `diag.Warningf` to create diagnostics with severity.**

**RULE: Compare errors with `errors.Is` or `errors.As` against a sentinel or typed error. Never branch on `err.Error()` string content.** The SDK exposes sentinels like `apierr.ErrNotFound` and `apierr.ErrResourceDoesNotExist`; the CLI has its own helpers like `isResourceGone`. String-matching error messages breaks the moment the upstream wording changes.

GOOD:

```go
import "github.com/databricks/databricks-sdk-go/apierr"

if errors.Is(err, apierr.ErrResourceDoesNotExist) {
	return nil
}
```

BAD:

```go
if err != nil && strings.Contains(err.Error(), "does not exist") {
	return nil
}
```

# CLI UX and validation

**RULE: Reject incompatible inputs early with an actionable error. Never silently ignore a flag or config field the current mode can't honor.** If a flag is incompatible with another flag or with a mode, return an error at flag-parse or validation time that tells the user which flag pair is at fault and what to do. If a config field applies only to certain resource types or engines, return a validation error, not a warning that gets lost in log output.

GOOD:

```go
if opts.Bind && opts.Resource != "dashboards" {
	return fmt.Errorf("--bind is only supported for dashboards, got %q", opts.Resource)
}
```

BAD:

```go
if opts.Bind && opts.Resource != "dashboards" {
	// silently drop the flag; user can't tell why nothing happened
}
```
