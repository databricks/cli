This file provides guidance to AI assistants when working with code in this repository.

Also consider the `.agents` directory.
- `rules/*.md` can be intelligently loaded through its YAML frontmatter
- `skills/*/SKILL.md` contains specific instructions that can be pulled in when needed for specific tasks (consult SKILL.md frontmatter)

# Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Declarative Automation Bundles (DABs), formerly known as Databricks Asset Bundles. The project is written in Go and follows a modular architecture.

# General Rules

When moving code from one place to another, please don't unnecessarily change the code or omit parts.

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

Use `git rm` to remove and `git mv` to rename files instead of directly modifying files on FS.

If asked to rebase, always prefix each git command with appropriate settings so that it never launches interactive editor:
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

- Use `make test-update` to regenerate acceptance test outputs after changes
- The CLI binary supports both `databricks` and `pipelines` command modes based on executable name
- Comments should explain "why", not "what" — reviewers consistently reject comments that merely restate the code

# Common Mistakes

- Do NOT add dependencies without checking license compatibility.
- Do NOT use `os.Exit()` outside of `main.go`.
- Do NOT remove or skip failing tests to fix CI — fix the underlying issue.
- Do NOT leave debug print statements (`fmt.Println`, `log.Printf` for debugging) in committed code — always scrub before committing.

# Error Handling

- Wrap errors with context: `fmt.Errorf("failed to deploy %s: %w", name, err)`
- Use `logdiag.LogDiag` / `logdiag.LogError` for logging diagnostics.
- Return early on errors; avoid deeply nested if-else chains.
- Use `diag.Errorf` / `diag.Warningf` to create diagnostics with severity.
