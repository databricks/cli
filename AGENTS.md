This file provides guidance to AI assistants when working with code in this repository.

# Project Overview

This is the Databricks CLI, a command-line interface for interacting with Databricks workspaces and managing Databricks Assets Bundles (DABs). The project is written in Go and follows a modular architecture.

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
Use "git rm" to remove and "git mv" to rename files instead of directly modifying files on FS.

If asked to rebase, always prefix each git command with appropriate settings so that it never launches interactive editor.
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git fetch origin main &&
GIT_EDITOR=true GIT_SEQUENCE_EDITOR=true VISUAL=true GIT_PAGER=cat git rebase origin/main

# Architecture

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

# Code Style and Patterns

## Go

Please make sure code that you author is consistent with the codebase and concise.

The code should be self-documenting based on the code and function names.

Functions should be documented with a doc comment as follows:

// SomeFunc does something.
func SomeFunc() {
	...
}

Note how the comment starts with the name of the function and is followed by a period.

Avoid redundant and verbose comments. Use terse comments and only add comments if it complements, not repeats the code.

Focus on making implementation as small and elegant as possible. Avoid unnecessary loops and allocations. If you see an opportunity of making things simpler by dropping or relaxing some requirements, ask user about the trade-off.

Use modern idiomatic Golang features (version 1.24+). Specifically:
 - Use for-range for integer iteration where possible. Instead of for i:=0; i < X; i++ {} you must write for i := range X{}.
 - Use builtin min() and max() where possible (works on any type and any number of values).
 - Do not capture the for-range variable, since go 1.22 a new copy of the variable is created for each loop iteration.

### Configuration Patterns
- Bundle config uses `dyn.Value` for dynamic typing
- Config loading supports includes, variable interpolation, and target overrides
- Schema generation is automated from Go struct tags

## Python

When writing Python scripts, we bias for conciseness. We think of Python in this code base as scripts.
 - use Python 3.11
 - Do not catch exceptions to make nicer messages, only catch if you can add critical information
 - use pathlib.Path in almost all cases over os.path unless it makes code longer
 - Do not add redundant comments.
 - Try to keep your code small and the number of abstractions low.
 - After done, format you code with "ruff format -n <path>"
 - Use "#!/usr/bin/env python3" shebang.

# Testing

### Test Types
- **Unit tests**: Standard Go tests alongside source files
- **Integration tests**: `integration/` directory, requires live Databricks workspace
- **Acceptance tests**: `acceptance/` directory, uses mock HTTP server

Each file like process_target_mode_test.go should have a corresponding test file
like process_target_mode_test.go. If you add new functionality to a file,
the test file should be extended to cover the new functionality.

Tests should look like the following:

package mutator_test

func TestApplySomeChangeReturnsDiagnostics(t *testing.T) {
	...
}

func TestApplySomeChangeFixesThings(t *testing.T) {
	ctx := context.Background()
	b, err := ...some operation...
	require.NoError(t, err)
	...
	assert.Equal(t, ...)
}

Notice that:
- Tests are often in the same package but suffixed wit _test.
- The test names are prefixed with Test and are named after the function or module they are testing.
- 'require' and 'require.NoError' are used to check for things that would cause the rest of the test case to fail.
- 'assert' is used to check for expected values where the rest of the test is not expected to fail.

When writing tests, please don't include an explanation in each
test case in your responses. I am just interested in the tests.

### Acceptance Tests
- Located in `acceptance/` with nested directory structure
- Each test directory contains `databricks.yml`, `script`, and `output.txt`
- Run with `go test ./acceptance -run TestAccept/bundle/<path>/<to>/<folder> -tail -test.v`
- Use `-update` flag to regenerate expected output files
- When you see the test fails because it has an old output, just run it one more time with an `-update` flag instead of changing the `output.txt` directly

# Logging

Use the following for logging:

```
import "github.com/databricks/cli/libs/log"

log.Infof(ctx, "...")
log.Debugf(ctx, "...")
log.Warnf(ctx, "...")
log.Errorf(ctx, "...")
```

Note that the 'ctx' variable here is something that should be passed in as
an argument by the caller. We should not use context.Background() like we do in tests.

Use cmdio.LogString to print to stdout:

```
import "github.com/databricks/cli/libs/cmdio"

cmdio.LogString(ctx, "...")
```

# Specific File Guides

## databricks_template_schema.json

A databricks_template_schema.json file is used to configure bundle templates.

Below is a good reference template:

{
    "welcome_message": "\nWelcome to the dbt template for Databricks Asset Bundles!\n\nA workspace was selected based on your current profile. For information about how to change this, see https://docs.databricks.com/dev-tools/cli/profiles.html.\nworkspace_host: {{workspace_host}}",
    "properties": {
        "project_name": {
            "type": "string",
            "pattern": "^[A-Za-z_][A-Za-z0-9-_]+$",
            "pattern_match_failure_message": "Name must consist of letters, numbers, dashes, and underscores.",
            "default": "dbt_project",
            "description": "\nPlease provide a unique name for this project.\nproject_name",
            "order": 1
        },
        "http_path": {
            "type": "string",
            "pattern": "^/sql/.\\../warehouses/[a-z0-9]+$",
            "pattern_match_failure_message": "Path must be of the form /sql/1.0/warehouses/<warehouse id>",
            "description": "\nPlease provide the HTTP Path of the SQL warehouse you would like to use with dbt during development.\nYou can find this path by clicking on \"Connection details\" for your SQL warehouse.\nhttp_path [example: /sql/1.0/warehouses/abcdef1234567890]",
            "order": 2
        },
        "default_catalog": {
            "type": "string",
            "default": "{{default_catalog}}",
            "pattern": "^\\w*$",
            "pattern_match_failure_message": "Invalid catalog name.",
            "description": "\nPlease provide an initial catalog{{if eq (default_catalog) \"\"}} (leave blank when not using Unity Catalog){{end}}.\ndefault_catalog",
            "order": 3
        },
        "personal_schemas": {
            "type": "string",
            "description": "\nWould you like to use a personal schema for each user working on this project? (e.g., 'catalog.{{short_name}}')\npersonal_schemas",
            "enum": [
                "yes, use a schema based on the current user name during development",
                "no, use a shared schema during development"
            ],
            "order": 4
        },
        "shared_schema": {
            "skip_prompt_if": {
                "properties": {
                    "personal_schemas": {
                        "const": "yes, use a schema based on the current user name during development"
                    }
                }
            },
            "type": "string",
            "default": "default",
            "pattern": "^\\w+$",
            "pattern_match_failure_message": "Invalid schema name.",
            "description": "\nPlease provide an initial schema during development.\ndefault_schema",
            "order": 5
        }
    },
    "success_message": "\n📊 Your new project has been created in the '{{.project_name}}' directory!\nIf you already have dbt installed, just type 'cd {{.project_name}}; dbt init' to get started.\nRefer to the README.md file for full \"getting started\" guide and production setup instructions.\n"
}

Notice that:
- The welcome message has the template name.
- By convention, property messages  include the property name after a newline, e.g. default_catalog above has a description that says "\nPlease provide an initial catalog [...].\ndefault_catalog",
- Each property defines a variable that is used for the template.
- Each property has a unique 'order' value that increments by 1 with each property.
- Enums use 'type: "string' and have an 'enum' field with a list of possible values.
- Helpers such as {{default_catalog}} and {{short_name}} can be used within property descriptors.
- Properties can be referenced in messages and descriptions using {{.property_name}}. {{.project_name}} is an example.

# Development Tips

- Run `make checks fmt lint` before committing
- Use `make test-update` to regenerate acceptance test outputs after changes
- The CLI binary supports both `databricks` and `pipelines` command modes based on executable name
- Resource definitions in `bundle/config/resources/` are auto-generated from OpenAPI specs
