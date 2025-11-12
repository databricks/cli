# Implementation Plan: `validate_data_app` Tool in Go

## Overview

Port the `validate_data_app` MCP tool from Rust (edda_mcp) to Go (go-mcp). This tool validates scaffolded applications by running build, type checking, and tests in an isolated sandbox environment.

## Reference Implementation

**Rust Location**: `edda/edda_mcp/src/providers/io.rs` (lines 300-334)

**Key Features**:
- Runs validation checks in isolated sandbox (using sandbox abstraction layer)
- Tracks project state with BLAKE3 checksums in `.edda_state` file
- Supports pluggable validation strategies (default tRPC vs custom)
- Provides detailed error reporting with stdout/stderr capture
- Idempotent state transitions (no partial updates on failure)
- Uses sandbox abstraction to be implementation-agnostic (Dagger/local)

## Implementation Steps

### 1. Add State Management (`pkg/state/`)

**New Package**: `pkg/state/state.go`

**Components**:
- `ProjectState` struct with states: `Scaffolded`, `Validated`
- `ProjectStateData` struct with:
  - `State` (enum: scaffolded/validated)
  - `Timestamp` (ISO 8601)
  - `Checksum` (BLAKE3 hash, optional for validated state)
  - `Version` (schema version)
- `LoadState(workDir string) (*ProjectStateData, error)` - Load from `.edda_state`
- `SaveState(workDir string, state *ProjectStateData) error` - Persist as JSON
- `ComputeChecksum(workDir string) (string, error)` - BLAKE3 hash of source files

**Checksum Logic**:
- **Include**: `client/`, `server/`, `package.json`
- **File Types**: `.ts`, `.tsx`, `.js`, `.jsx`, `.json`, `.css`, `.html`, `.yaml`, `.yml`
- **Exclude Dirs**: `node_modules`, `dist`, `.git`, `build`, `coverage`
- **Algorithm**: BLAKE3 (cryptographic hash)

**Dependencies**:
- Add `github.com/zeebo/blake3` to `go.mod`

### 2. Add Validation Strategy Pattern (`pkg/providers/io/validation.go`)

**New File**: `pkg/providers/io/validation.go`

**Interface**:
```go
type ValidationStrategy interface {
    Validate(ctx context.Context, sandbox sandbox.Sandbox) (*ValidationResult, error)
}
```

**Implementations**:

**a) ValidationTRPC** (default strategy):
- `NewValidationTRPC() ValidationStrategy`
- Sequence:
  1. `runBuild()` - Execute `npm run build` in `/app`
  2. `runTypeCheck()` - Execute `npx tsc --noEmit` in `/app/client`
  3. `runTests()` - Execute `npm test` in `/app`
- Short-circuits on first failure

**b) ValidationCmd** (custom command):
- `NewValidationCmd(command string) ValidationStrategy`
- Runs: `cd /app && {command}`
- For advanced users with custom validation workflows

**Result Types**:
```go
type ValidationResult struct {
    Success bool
    Message string
    Details *ValidationDetails
}

type ValidationDetails struct {
    ExitCode int
    Stdout   string
    Stderr   string
}
```

**Factory**:
```go
func NewValidationStrategy(config *config.ValidationConfig) ValidationStrategy {
    if config != nil && config.CustomCommand != "" {
        return NewValidationCmd(config.CustomCommand)
    }
    return NewValidationTRPC()
}
```

### 3. Implement Validation Checks

**Helper Functions** (in `validation.go`):

**a) `runBuild(ctx, sandbox) (*ValidationDetails, error)`**:
- Command: `cd /app && npm run build`
- Checks TypeScript/JavaScript compilation
- Returns details with exit code, stdout, stderr

**b) `runTypeCheck(ctx, sandbox) (*ValidationDetails, error)`**:
- Command: `cd /app/client && npx tsc --noEmit`
- Validates TypeScript type safety
- Detects type errors without emitting files

**c) `runTests(ctx, sandbox) (*ValidationDetails, error)`**:
- Command: `cd /app && npm test`
- Executes test suite
- Returns failure if tests don't pass

**Error Handling**:
- Capture both stdout and stderr
- Non-zero exit code = validation failure
- Include full output in result details

### 4. Update IO Provider (`pkg/providers/io/provider.go`)

**Add Tool Registration**:
```go
func (p *Provider) RegisterTools(server *mcp.Server) error {
    // ... existing tools ...

    server.AddTool(mcp.Tool{
        Name: "validate_data_app",
        Description: "Validate a project by copying files to a sandbox and running validation checks. Project should be scaffolded first. Returns validation result with success status and details.",
        InputSchema: generateSchema(ValidateDataAppInput{}),
    }, p.handleValidateDataApp)

    return nil
}
```

**Input Schema**:
```go
type ValidateDataAppInput struct {
    WorkDir string `json:"work_dir" jsonschema:"required,description=Absolute path to the work directory to validate"`
}
```

**Handler Implementation** (`handleValidateDataApp`):

**Workflow**:
1. **Parse Input**: Decode `work_dir` from arguments
2. **Validate Path**:
   - Ensure absolute path (reject relative paths)
   - Check directory exists
3. **Load Strategy**: `NewValidationStrategy(p.config.Validation)`
4. **Create Sandbox**:
   - Docker image: `node:20-alpine3.22` (configurable)
   - Timeout: 600 seconds
   - Logger: Silent mode
5. **Setup Container**:
   - Create `/app` directory
   - Mount `work_dir` → `/app`
   - Propagate env vars: `DATABRICKS_HOST`, `DATABRICKS_TOKEN`, `DATABRICKS_WAREHOUSE_ID`
6. **Run Validation**: `strategy.Validate(ctx, sandbox)`
7. **Handle Success**:
   - Compute checksum: `state.ComputeChecksum(workDir)`
   - Update state: `ProjectState = Validated`
   - Save state: `state.SaveState(workDir, stateData)`
   - Return success result
8. **Handle Failure**:
   - Do NOT update state
   - Return failure with details

**Return Format**:
```go
return &mcp.CallToolResult{
    Content: []mcp.Content{
        mcp.TextContent(result.String()),
    },
}
```

**Result Display** (implement `String()` on `ValidationResult`):
- Success: `"Validation passed: {message}"`
- Failure: `"Validation failed: {message}\nExit code: {code}\nStdout: {stdout}\nStderr: {stderr}"`

### 5. Update Configuration (`pkg/config/config.go`)

**Add Validation Config**:
```go
type ValidationConfig struct {
    DockerImage   string `json:"docker_image,omitempty"`   // Default: "node:20-alpine3.22"
    CustomCommand string `json:"custom_command,omitempty"` // Optional custom validation
}

type IOConfig struct {
    Validation *ValidationConfig `json:"validation,omitempty"`
}

type Config struct {
    // ... existing fields ...
    IO *IOConfig `json:"io,omitempty"`
}
```

**Default Values**:
- If `IO.Validation` is nil, use default tRPC strategy
- If `IO.Validation.DockerImage` is empty, use `"node:20-alpine3.22"`

### 6. Write Tests

**Unit Tests** (`pkg/providers/io/validation_test.go`):
- Test `ComputeChecksum` with known file structure
- Test `LoadState` / `SaveState` roundtrip
- Test validation strategy selection

**Integration Tests** (`test/integration/validate_test.go`):
- **Test 1**: Successful validation (all checks pass)
  - Scaffold default template
  - Run validate_data_app
  - Assert success = true
  - Verify `.edda_state` updated to Validated
- **Test 2**: Type error detection
  - Scaffold template
  - Inject type error in client code
  - Run validate_data_app
  - Assert failure with type check details
- **Test 3**: Failing tests
  - Scaffold template
  - Modify test to fail
  - Run validate_data_app
  - Assert failure with test output

### 7. Update Documentation (`go-mcp/CLAUDE.md`)

**Add to IO Provider Section** (around line 105):
```markdown
**IO Provider** (pkg/providers/io)
- Tools: scaffold_data_app, validate_data_app
- Embeds templates at compile time
- Sets WorkDir in session for workspace tools
- Validates applications in Dagger sandbox
```

**Add New Section**:
```markdown
### Validation (pkg/providers/io/validation.go)

**Validation Strategies**:
- **ValidationTRPC** (default): npm build → tsc type check → npm test
- **ValidationCmd** (custom): User-defined command

**State Tracking**:
- `.edda_state` file tracks project lifecycle (Scaffolded → Validated)
- BLAKE3 checksums detect source file changes
- State only updated on successful validation (idempotent)
```

## Dependencies

**New Dependencies**:
- `github.com/zeebo/blake3` - BLAKE3 hashing

**Existing Dependencies** (already in go-mcp):
- Dagger SDK for sandbox
- Official MCP SDK
- Standard library for file operations

## Testing Checklist

- [ ] State management unit tests
- [ ] Checksum computation with various file structures
- [ ] Validation strategy selection logic
- [ ] Successful validation integration test
- [ ] Type error detection integration test
- [ ] Failing test detection integration test
- [ ] State persistence after validation
- [ ] Error handling for missing/invalid paths

## Success Criteria

1. Tool successfully validates scaffolded applications
2. All three checks (build, type, test) execute correctly
3. State transitions from Scaffolded → Validated on success
4. Checksums detect source file changes
5. Integration tests pass with >80% coverage
6. Error messages provide actionable debugging info

## References

- **Rust Implementation**: `edda/edda_mcp/src/providers/io.rs:300-334`
- **Rust Tests**: `edda/edda_mcp/tests/validate_project.rs`
- **State Module**: `edda/edda_mcp/src/state.rs`
- **Validation Trait**: `edda/edda_mcp/src/validation/mod.rs:372`
