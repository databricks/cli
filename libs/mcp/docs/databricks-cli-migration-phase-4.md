# Phase 4: Testing & Build Integration

**bd Issue**: `parity-42` (task)
**Status**: Open | **Priority**: P0
**Depends on**: parity-41 (Phase 3)
**Blocks**: parity-43 (Phase 5)

## Overview

This phase adapts the test suite to Databricks CLI patterns and integrates with the CLI's build system and CI/CD pipelines.

## Duration

2-3 hours

## Prerequisites

- Phase 3 completed
- Code builds cleanly
- Infrastructure integrated

## Tasks

### 4.1 Copy Test Files

**Objective**: Migrate all test files from go-mcp

**Source → Target Mapping**:
```
app-mcp/pkg/providers/databricks/*_test.go
  → cli/libs/mcp/providers/databricks/*_test.go

app-mcp/pkg/providers/io/*_test.go
  → cli/libs/mcp/providers/io/*_test.go

app-mcp/pkg/providers/workspace/*_test.go
  → cli/libs/mcp/providers/workspace/*_test.go

app-mcp/pkg/sandbox/local/*_test.go
  → cli/libs/mcp/sandbox/local/*_test.go

app-mcp/pkg/trajectory/*_test.go
  → cli/libs/mcp/trajectory/*_test.go

app-mcp/pkg/session/*_test.go
  → cli/libs/mcp/session/*_test.go

app-mcp/pkg/errors/*_test.go
  → cli/libs/mcp/errors/*_test.go

app-mcp/pkg/pathutil/*_test.go
  → cli/libs/mcp/pathutil/*_test.go
```

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/app-mcp

# Copy all test files
find pkg -name "*_test.go" -type f | while read f; do
  target=$(echo "$f" | sed 's|^pkg|/Users/fabian.jakobs/Workspaces/cli/libs/mcp|')
  cp "$f" "$target"
done

# Copy test fixtures and testdata
find pkg -type d -name "testdata" | while read d; do
  target=$(echo "$d" | sed 's|^pkg|/Users/fabian.jakobs/Workspaces/cli/libs/mcp|')
  mkdir -p "$target"
  cp -r "$d"/* "$target/"
done
```

**Verification**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
find libs/mcp -name "*_test.go" | wc -l
# Should match count from go-mcp
```

### 4.2 Update Test Import Paths

**Objective**: Update import paths in test files

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Update import paths in test files
find libs/mcp -name "*_test.go" -exec sed -i '' \
  's|github.com/appdotbuild/go-mcp/pkg/|github.com/databricks/cli/libs/mcp/|g' {} +

# Update mcp package reference
find libs/mcp -name "*_test.go" -exec sed -i '' \
  's|github.com/databricks/cli/libs/mcp/mcp|github.com/databricks/cli/libs/mcp/server|g' {} +

# Update logging imports
find libs/mcp -name "*_test.go" -exec sed -i '' \
  's|github.com/databricks/cli/libs/mcp/logging|github.com/databricks/cli/libs/log|g' {} +
```

**Verification**:
```bash
# No old imports
grep -r "github.com/appdotbuild/go-mcp" libs/mcp/*_test.go || echo "✓ Clean"
```

### 4.3 Adapt Test Patterns

**Objective**: Align tests with Databricks CLI testing conventions

**Common Patterns to Update**:

1. **Test Setup with Context**:
```go
// Before
func TestSomething(t *testing.T) {
    logger := logging.NewLogger("test", slog.LevelDebug)
    // ...
}

// After
func TestSomething(t *testing.T) {
    ctx := context.Background()
    // Logger will be retrieved from context in code
    // ...
}
```

2. **Mock Dependencies**:
```go
// Ensure mocks work with context-based patterns
type mockProvider struct {
    // ...
}

func (m *mockProvider) HandleTool(ctx context.Context, args map[string]interface{}) (*Result, error) {
    // Implementation
}
```

3. **Test Helpers**:
```go
// Create test helpers for common setup
func testContext(t *testing.T) context.Context {
    ctx := context.Background()
    // Add test logger if needed
    return ctx
}

func testSession(t *testing.T) *session.Session {
    return session.NewSession()
}
```

**Files Requiring Manual Updates**:
- Tests that use logging directly
- Tests that create configs
- Tests that use session state
- Integration tests

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Run tests to find issues
go test ./libs/mcp/... 2>&1 | tee /tmp/test-errors.txt

# Review and fix errors systematically
```

### 4.4 Run Test Suite

**Objective**: Ensure all tests pass

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Run all MCP tests
go test ./libs/mcp/...

# Run with verbose output
go test -v ./libs/mcp/...

# Run with coverage
go test -cover ./libs/mcp/...

# Run with race detector
go test -race ./libs/mcp/...
```

**Test Categories**:
1. **Unit Tests**: Individual package tests
2. **Provider Tests**: Tool handler tests
3. **Sandbox Tests**: Local sandbox operations
4. **Integration Tests**: End-to-end workflows (if any)

**Fix Common Issues**:
- Context not passed correctly
- Logger initialization
- Config setup in tests
- Session state management
- Path handling differences

**Success Criteria**:
- All tests pass
- No race conditions
- Coverage maintained (>80%)

### 4.5 Integration Tests

**Objective**: Create or adapt integration tests

**Test Scenarios**:
1. **Command Execution**:
```go
func TestMCPCommandStart(t *testing.T) {
    // Test that 'databricks apps mcp start' can be executed
    // (May require mocking or test environment)
}
```

2. **Provider Registration**:
```go
func TestProviderRegistration(t *testing.T) {
    ctx := context.Background()
    cfg := &config.Config{
        WarehouseID: "test-warehouse",
    }
    server := mcpserver.NewServer(cfg, ctx)
    // Verify providers are registered
}
```

3. **Tool Execution**:
```go
func TestToolExecution(t *testing.T) {
    // Test executing a tool through the full stack
}
```

**Create Integration Test Directory**:
```bash
mkdir -p libs/mcp/integration
```

**Create Integration Test File**:
```go
// libs/mcp/integration/server_test.go
// +build integration

package integration

import (
    "context"
    "testing"

    mcpserver "github.com/databricks/cli/libs/mcp/server"
    "github.com/databricks/cli/libs/mcp/config"
)

func TestServerInitialization(t *testing.T) {
    ctx := context.Background()
    cfg := &config.Config{
        WarehouseID: "test-warehouse",
    }

    server := mcpserver.NewServer(cfg, ctx)
    if server == nil {
        t.Fatal("Failed to create server")
    }

    // Test server capabilities
}
```

**Run Integration Tests**:
```bash
go test -tags=integration ./libs/mcp/integration/...
```

### 4.6 Update Makefile

**Objective**: Integrate MCP tests into CLI build system

**Edit `/Users/fabian.jakobs/Workspaces/cli/Makefile`**:

Add MCP packages to test targets (if not already included by wildcards):
```makefile
# Existing packages should already include libs/mcp via ./libs/...
# Verify by checking PACKAGES variable

# Add MCP-specific targets if needed
.PHONY: test-mcp
test-mcp:
	go test ./libs/mcp/...

.PHONY: test-mcp-integration
test-mcp-integration:
	go test -tags=integration ./libs/mcp/integration/...

.PHONY: test-mcp-coverage
test-mcp-coverage:
	go test -cover -coverprofile=coverage-mcp.out ./libs/mcp/...
	go tool cover -html=coverage-mcp.out -o coverage-mcp.html
```

**Verification**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Test standard test target
make test

# Test MCP-specific target
make test-mcp
```

### 4.7 CI/CD Integration

**Objective**: Ensure CI/CD pipelines include MCP tests

**Files to Check**:
- `.github/workflows/*.yml` (if using GitHub Actions)
- CI configuration files

**Verification**:
- MCP packages included in test runs
- No special configuration needed (should be automatic via `./libs/...`)

**Note**: Most CI systems will automatically pick up new packages, but verify this.

### 4.8 Test Coverage Analysis

**Objective**: Verify test coverage is maintained

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Generate coverage report
go test -coverprofile=coverage-mcp.out ./libs/mcp/...

# View coverage summary
go tool cover -func=coverage-mcp.out

# Generate HTML report
go tool cover -html=coverage-mcp.out -o coverage-mcp.html

# View in browser
open coverage-mcp.html
```

**Coverage Targets**:
- Overall: >80%
- Critical paths (providers, sandbox): >90%
- Utility packages: >85%

**If Coverage Low**:
- Add missing test cases
- Test error paths
- Add integration tests

### 4.9 Benchmark Tests

**Objective**: Ensure performance is maintained

**Run Benchmarks** (if any exist):
```bash
cd /Users/fabian.jakobs/Workspaces/cli

# Run benchmarks
go test -bench=. ./libs/mcp/...

# With memory allocation stats
go test -bench=. -benchmem ./libs/mcp/...
```

**Key Areas**:
- Sandbox operations
- File I/O
- Template rendering
- Provider tool execution

### 4.10 Create Testing Integration Commit

**Objective**: Commit the test integration

**Commit Message**:
```
Integrate MCP tests with Databricks CLI build system (migration step 3/3)

Migrate and adapt test suite from go-mcp:

- Copied all test files with updated import paths
- Adapted test patterns to CLI conventions (context-based)
- Integrated with Makefile and gotestsum
- All tests passing (unit, integration, race detector)
- Test coverage maintained at >80%

Test Results:
- Unit tests: PASS
- Integration tests: PASS
- Race detector: PASS
- Coverage: 82% (libs/mcp)

Phase 4 of 5: Testing & Build Integration
Status: All tests passing, ready for documentation

Related:
- Phase 3: Infrastructure Integration (complete)
- Phase 5: Documentation & Finalization (next)
```

**Steps**:
```bash
cd /Users/fabian.jakobs/Workspaces/cli
git add -A
git commit -F- <<'EOF'
Integrate MCP tests with Databricks CLI build system (migration step 3/3)

[Full message above]
EOF
```

## Deliverables

1. ✅ Test files copied and import paths updated
2. ✅ Test patterns adapted to CLI conventions
3. ✅ All unit tests passing
4. ✅ Integration tests created/adapted
5. ✅ Makefile updated with MCP targets
6. ✅ CI/CD integration verified
7. ✅ Test coverage analyzed and maintained
8. ✅ Benchmark tests run
9. ✅ Testing integration commit created

## Verification Steps

- [ ] All tests pass: `make test`
- [ ] MCP tests pass: `go test ./libs/mcp/...`
- [ ] Race detector clean: `go test -race ./libs/mcp/...`
- [ ] Coverage >80%: `go test -cover ./libs/mcp/...`
- [ ] Integration tests pass: `go test -tags=integration ./libs/mcp/integration/...`
- [ ] Makefile targets work: `make test-mcp`

## Test Summary

Expected test results after this phase:
```
PASS
coverage: 82.4% of statements
ok      github.com/databricks/cli/libs/mcp/providers/databricks    0.234s
ok      github.com/databricks/cli/libs/mcp/providers/io            0.456s
ok      github.com/databricks/cli/libs/mcp/providers/workspace     0.189s
ok      github.com/databricks/cli/libs/mcp/sandbox/local           0.345s
ok      github.com/databricks/cli/libs/mcp/trajectory              0.123s
ok      github.com/databricks/cli/libs/mcp/session                 0.089s
```

## Next Phase

**Phase 5: Documentation & Finalization** - Update documentation and prepare for merge
