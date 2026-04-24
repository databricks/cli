This file provides guidance to AI assistants when working with code in this repository.

# Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Declarative Automation Bundles (DABs), formerly known as Databricks Asset Bundles. The project is written in Go and follows a modular architecture.

# General Rules

**RULE: When moving code from one place to another, don't unnecessarily change or omit parts.** Keep refactors separate from content changes so reviewers can tell them apart.

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

# Common Mistakes

**RULE: Do not add dependencies without checking license compatibility.**

**RULE: Do not use `os.Exit()` outside of `main.go`.** `main.go` owns the exit path; calling `os.Exit()` elsewhere skips deferred cleanup and complicates testing.

**RULE: Do not remove or skip failing tests to fix CI.** Fix the underlying issue instead.

**RULE: Do not leave debug print statements in committed code.** `fmt.Println`, `log.Printf`, or similar. Always scrub before committing.

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
