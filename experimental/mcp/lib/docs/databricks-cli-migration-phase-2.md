# Phase 2: Code Structure Migration

**bd Issue**: `parity-40` (task)
**Status**: Open | **Priority**: P0
**Depends on**: parity-39 (Phase 1)
**Blocks**: parity-41 (Phase 3)

## Overview

This phase moves the go-mcp code into the Databricks CLI repository structure, updates import paths, and establishes the basic command hierarchy.

## Duration

2-3 hours

## Prerequisites

- Phase 1 completed
- `apps-mcp` branch ready
- Directory structure plan approved

## Tasks

### 2.1 Create Directory Structure

**Objective**: Set up the target directories in Databricks CLI

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Create command structure
mkdir -p cmd/apps
mkdir -p cmd/apps/mcp

# Create library structure
mkdir -p libs/mcp/server
mkdir -p libs/mcp/providers/databricks
mkdir -p libs/mcp/providers/io
mkdir -p libs/mcp/providers/workspace
mkdir -p libs/mcp/providers/deployment
mkdir -p libs/mcp/sandbox/local
mkdir -p libs/mcp/sandbox/dagger
mkdir -p libs/mcp/trajectory
mkdir -p libs/mcp/templates
mkdir -p libs/mcp/session
mkdir -p libs/mcp/pathutil
mkdir -p libs/mcp/fileutil

# Create internal structure
mkdir -p internal/mcp/templates
```

**Verification**:
```bash
tree -L 3 cmd/apps
tree -L 3 libs/mcp
```

### 2.2 Copy and Adapt Command Files

**Objective**: Migrate CLI entry point and command structure

**Source → Target Mapping**:
```
app-mcp/cmd/go-mcp/main.go → cli/cmd/apps/mcp/mcp.go
app-mcp/cmd/go-mcp/cli.go  → cli/cmd/apps/mcp/start.go
```

**Steps**:

1. **Create `cmd/apps/apps.go`** (new file):
```go
package apps

import (
	"github.com/databricks/cli/cmd/apps/mcp"
	"github.com/spf13/cobra"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "Databricks Apps commands",
		Long:  "Commands for managing and interacting with Databricks Apps",
	}

	cmd.AddCommand(mcp.New())
	return cmd
}
```

2. **Create `cmd/apps/mcp/mcp.go`**:
- Adapt main.go logic
- Change package to `mcp`
- Use CLI's cobra patterns
- Remove standalone main() function

3. **Create `cmd/apps/mcp/start.go`**:
- Adapt cli.go logic
- Implement RunE pattern from CLI
- Use libs/log for logging

4. **Create `cmd/apps/mcp/check.go`**:
- Migrate environment check logic
- Use CLI's output patterns

5. **Create `cmd/apps/mcp/config.go`**:
- Add config show/validate subcommands
- Integrate with CLI config system

**Key Changes**:
- Replace `fmt.Println` with proper CLI output (cmdio)
- Use context from cobra command
- Integrate with CLI's error handling

### 2.3 Copy Library Code

**Objective**: Migrate pkg/ code to libs/mcp/

**Migration Map**:
```
app-mcp/pkg/mcp/          → libs/mcp/server/
app-mcp/pkg/providers/    → libs/mcp/providers/
app-mcp/pkg/sandbox/      → libs/mcp/sandbox/
app-mcp/pkg/trajectory/   → libs/mcp/trajectory/
app-mcp/pkg/templates/    → libs/mcp/templates/
app-mcp/pkg/session/      → libs/mcp/session/
app-mcp/pkg/errors/       → libs/mcp/errors/       (temporary)
app-mcp/pkg/pathutil/     → libs/mcp/pathutil/
app-mcp/pkg/fileutil/     → libs/mcp/fileutil/
app-mcp/internal/templates/ → internal/mcp/templates/
```

**Files to Skip** (will be replaced):
```
app-mcp/pkg/config/       → Use CLI config patterns
app-mcp/pkg/logging/      → Use libs/log
```

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/app-mcp

# Copy providers
cp -r pkg/providers/* /Users/fabian.jakobs/Workspaces/cli/libs/mcp/providers/

# Copy sandbox
cp -r pkg/sandbox/* /Users/fabian.jakobs/Workspaces/cli/libs/mcp/sandbox/

# Copy MCP server implementation
cp -r pkg/mcp/* /Users/fabian.jakobs/Workspaces/cli/libs/mcp/server/

# Copy support libraries
cp -r pkg/trajectory /Users/fabian.jakobs/Workspaces/cli/libs/mcp/
cp -r pkg/templates /Users/fabian.jakobs/Workspaces/cli/libs/mcp/
cp -r pkg/session /Users/fabian.jakobs/Workspaces/cli/libs/mcp/
cp -r pkg/errors /Users/fabian.jakobs/Workspaces/cli/libs/mcp/
cp -r pkg/pathutil /Users/fabian.jakobs/Workspaces/cli/libs/mcp/
cp -r pkg/fileutil /Users/fabian.jakobs/Workspaces/cli/libs/mcp/

# Copy templates
cp -r internal/templates /Users/fabian.jakobs/Workspaces/cli/internal/mcp/
```

**Verification**:
- All files copied
- No config/ or logging/ in target
- Directory structure matches plan

### 2.4 Update Import Paths

**Objective**: Replace all import paths with Databricks CLI paths

**Changes Required**:
```
github.com/appdotbuild/go-mcp/pkg/mcp
  → github.com/databricks/cli/libs/mcp/server

github.com/appdotbuild/go-mcp/pkg/providers/databricks
  → github.com/databricks/cli/libs/mcp/providers/databricks

github.com/appdotbuild/go-mcp/pkg/providers/io
  → github.com/databricks/cli/libs/mcp/providers/io

github.com/appdotbuild/go-mcp/pkg/providers/workspace
  → github.com/databricks/cli/libs/mcp/providers/workspace

github.com/appdotbuild/go-mcp/pkg/providers/deployment
  → github.com/databricks/cli/libs/mcp/providers/deployment

github.com/appdotbuild/go-mcp/pkg/sandbox
  → github.com/databricks/cli/libs/mcp/sandbox

github.com/appdotbuild/go-mcp/pkg/trajectory
  → github.com/databricks/cli/libs/mcp/trajectory

github.com/appdotbuild/go-mcp/pkg/templates
  → github.com/databricks/cli/libs/mcp/templates

github.com/appdotbuild/go-mcp/pkg/session
  → github.com/databricks/cli/libs/mcp/session

github.com/appdotbuild/go-mcp/pkg/errors
  → github.com/databricks/cli/libs/mcp/errors

github.com/appdotbuild/go-mcp/pkg/pathutil
  → github.com/databricks/cli/libs/mcp/pathutil

github.com/appdotbuild/go-mcp/pkg/fileutil
  → github.com/databricks/cli/libs/mcp/fileutil
```

**Script to Update**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Update all go files
find cmd/apps/mcp libs/mcp internal/mcp -name "*.go" -type f -exec sed -i '' \
  's|github.com/appdotbuild/go-mcp/pkg/|github.com/databricks/cli/libs/mcp/|g' {} +

# Update mcp package reference
find cmd/apps/mcp libs/mcp internal/mcp -name "*.go" -type f -exec sed -i '' \
  's|github.com/databricks/cli/libs/mcp/mcp|github.com/databricks/cli/libs/mcp/server|g' {} +
```

**Verification**:
```bash
grep -r "github.com/appdotbuild/go-mcp" cmd/apps/mcp libs/mcp internal/mcp
# Should return no results
```

### 2.5 Register Apps Command

**Objective**: Wire up the new command in the CLI

**Edit `cmd/cmd.go`**:
```go
import (
	// ... existing imports ...
	"github.com/databricks/cli/cmd/apps"
)

func New(ctx context.Context) *cobra.Command {
	cli := root.New(ctx)

	// ... existing commands ...

	// Add apps command
	cli.AddCommand(apps.New())

	// ... rest of code ...
}
```

**Verification**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
go build ./cmd/cli
./cli apps --help
./cli apps mcp --help
```

### 2.6 Update Dependencies

**Objective**: Add new dependencies to go.mod

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Add new dependencies
go get github.com/modelcontextprotocol/go-sdk@v1.1.0
go get github.com/zeebo/blake3@v0.2.4

# Update Databricks SDK if needed
go get github.com/databricks/databricks-sdk-go@v0.90.0

# Tidy dependencies
go mod tidy
```

**Verification**:
```bash
go mod verify
grep "modelcontextprotocol" go.mod
grep "blake3" go.mod
```

### 2.7 Initial Build Test

**Objective**: Verify code compiles (may have errors to fix in Phase 3)

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Try to build
go build ./cmd/cli

# List compilation errors if any
go build ./cmd/cli 2>&1 | tee /tmp/build-errors.txt

# Analyze error patterns
grep "undefined:" /tmp/build-errors.txt | sort | uniq
```

**Expected Issues** (to be fixed in Phase 3):
- Logging references (pkg/logging → libs/log)
- Config references (pkg/config → CLI patterns)
- Import errors
- Type mismatches

**Success Criteria**:
- Code is copied and organized correctly
- Import paths updated
- Dependencies added
- Compilation errors are only infrastructure-related (logging, config)

### 2.8 Create Initial Commit

**Objective**: Commit the structural migration

**Commit Message**:
```
Add apps mcp command structure (migration step 1/3)

Migrate go-mcp standalone CLI into Databricks CLI as `databricks apps mcp`.

This commit includes:
- Command structure under cmd/apps/mcp
- Library code under libs/mcp
- Updated import paths to github.com/databricks/cli
- New dependencies: go-sdk, blake3

Source: go-mcp standalone repo (21 commits)
Original commits: 86db47a through 441f736

Phase 2 of 5: Code Structure Migration
Status: Compiles with known issues (logging, config integration needed)

Related:
- Phase 1: Repository Setup & Analysis (complete)
- Phase 3: Infrastructure Integration (next)
```

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
git add cmd/apps libs/mcp internal/mcp go.mod go.sum
git commit -F- <<'EOF'
Add apps mcp command structure (migration step 1/3)

[Full message above]
EOF
```

## Deliverables

1. ✅ Directory structure created
2. ✅ Command files migrated
3. ✅ Library code migrated
4. ✅ Import paths updated
5. ✅ Apps command registered
6. ✅ Dependencies added
7. ✅ Initial build attempted
8. ✅ Migration commit created

## Verification Steps

- [ ] All directories exist
- [ ] No app-mcp import paths remain
- [ ] go.mod includes new dependencies
- [ ] `git status` shows new files
- [ ] Build errors are infrastructure-related only
- [ ] Command structure visible: `./cli apps mcp --help`

## Known Issues (to be addressed in Phase 3)

1. Logging references need libs/log migration
2. Config initialization needs CLI integration
3. Session management needs context integration
4. Error handling may need standardization

## Next Phase

**Phase 3: Infrastructure Integration** - Replace go-mcp infrastructure with CLI equivalents
