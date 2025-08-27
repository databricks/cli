# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Databricks Assets Bundles (DABs). The project is written in Go and follows a modular architecture.

## Development Commands

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

## Architecture

### Core Components

**cmd/** - CLI command structure using Cobra framework
- `cmd/cmd.go` - Main command setup and subcommand registration
- `cmd/bundle/` - Bundle-related commands (deploy, validate, etc.)
- `cmd/workspace/` - Workspace API commands (auto-generated)
- `cmd/account/` - Account-level API commands (auto-generated)

**bundle/** - Core bundle functionality for Databricks Asset Bundles
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

**Direct vs Terraform Deployment**: The CLI supports two deployment modes controlled by `DATABRICKS_CLI_DEPLOYMENT` environment variable:
- `terraform` (default) - Uses Terraform for resource management
- `direct` - Direct API calls without Terraform

## Testing

### Test Types
- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

### Acceptance Tests
- Located in `acceptance/` with nested directory structure
- Each test directory contains `databricks.yml`, `script`, and `output.txt`
- Run with `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder> -tail -test.v`
- Use `-update` flag to regenerate expected output files
- When you see the test fails because it has an old output, just run it one more time with an `-update` flag instead of changing the `output.txt` directly

## Code Patterns

### Error Handling
- Bundle operations return diagnostics rather than plain errors
- Authentication errors should be wrapped with context

### Configuration
- Bundle config uses `dyn.Value` for dynamic typing
- Config loading supports includes, variable interpolation, and target overrides
- Schema generation is automated from Go struct tags

### File Operations
- Always use `vfs.Path` and `filer.Filer` interfaces rather than direct file operations
- Support for local filesystem, DBFS, and Workspace File System

## Development Tips

- Run `make checks fmt lint` before committing
- Use `make test-update` to regenerate acceptance test outputs after changes
- The CLI binary supports both `databricks` and `pipelines` command modes based on executable name
- Resource definitions in `bundle/config/resources/` are auto-generated from OpenAPI specs
