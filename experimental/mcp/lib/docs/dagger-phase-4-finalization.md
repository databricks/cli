# Dagger Phase 4: Documentation, 

**Issue**: parity-47
**Parent**: parity-2 (Phase 1: Dagger Sandbox Implementation)
**Depends On**: parity-44, parity-45, parity-46
**Estimated Time**: 3 days

## Objective

Complete the Dagger implementation with comprehensive documentation, and final verification.

## Prerequisites

- Phases 1-3 completed (parity-44, parity-45, parity-46)
- All operations working and tested
- Integration tests passing

## Implementation Steps

### Step 1: Documentation Updates

#### 1.1 Update CLAUDE.md

Add comprehensive Dagger documentation to `CLAUDE.md`:

**Location**: After the "Sandbox (pkg/sandbox)" section

```markdown
### Dagger Implementation (pkg/sandbox/dagger)

**Status**: ✅ Fully Implemented

Containerized sandbox using Dagger SDK for isolated, reproducible validation.

**Core Operations:**
- `Exec(cmd)` - Execute shell commands in container
- `WriteFile(path, content)` - Write single file
- `WriteFiles(files)` - Bulk write (optimized with temp dir pattern)
- `ReadFile(path)` - Read file contents
- `DeleteFile(path)` - Remove file
- `ListDirectory(path)` - List directory entries
- `SetWorkdir(path)` - Change working directory

**Advanced Operations:**
- `ExportDirectory(container, host)` - Export artifacts to host
- `RefreshFromHost(host, container)` - Import files from host
- `Fork()` - Clone sandbox state for parallel operations
- `Close()` - Cleanup Dagger resources

**Environment Support:**
- `WithEnv(key, value)` - Set single environment variable
- `WithEnvs(map)` - Set multiple environment variables
- Automatic propagation of DATABRICKS_* credentials

**Configuration:**
```go
type Config struct {
    Image          string // Default: node:20-alpine3.22
    ExecuteTimeout int    // Default: 600 seconds
    BaseDir        string // Host directory
}
```

**Usage Example:**
```go
ctx := context.Background()
sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
    Image: "node:20-alpine3.22",
    ExecuteTimeout: 600,
})
if err != nil {
    return err
}
defer sb.Close()

// Propagate environment
sb.WithEnv("DATABRICKS_HOST", os.Getenv("DATABRICKS_HOST"))

// Sync project files
err = sb.RefreshFromHost(ctx, "/path/to/project", "/workspace")

// Run validation
result, err := sb.Exec(ctx, "npm test")
```

**Security Features:**
- Process isolation via containers
- Configurable Docker images
- Execution timeouts
- No privileged access
- Clean state between validations

**Performance:**
- Bulk file operations optimized with temp directories

**Fallback Strategy:**
- Dagger enabled by default (`use_dagger: true`)
- Falls back to local sandbox if Docker unavailable
- Logs warning and continues gracefully

**Testing:**
- Unit tests: `pkg/sandbox/dagger/dagger_test.go`
- Integration tests: `test/integration/dagger_validation_test.go`
- Run with: `go test -v ./pkg/sandbox/dagger/` (requires Docker)

**Reference Implementation:**
- Rust: `../agent/edda/edda_sandbox/src/dagger.rs`
- Behavioral parity achieved
```

#### 1.2 Update README.md

Add Dagger section to `README.md`:

**Location**: After "Features" section

```markdown
## Sandboxed Validation

go-mcp supports two sandbox modes for secure validation:

### Dagger (Default)

Containerized validation using [Dagger](https://dagger.io/):
- **Isolated**: Each validation runs in a fresh container
- **Reproducible**: Consistent Node.js environment (node:20-alpine3.22)
- **Secure**: No access to host filesystem outside project
- **Portable**: Works the same on any machine with Docker

**Requirements:**
- Docker daemon running
- Dagger SDK (installed automatically via `go get`)

**Configuration:**
```json
{
  "io_config": {
    "validation": {
      "use_dagger": true,
      "docker_image": "node:20-alpine3.22",
      "timeout": 600
    }
  }
}
```

### Local Sandbox

Default mode using host filesystem:
- Faster for rapid development
- No Docker required
- Less isolated (uses host Node.js)

**Enable local-only:**
```json
{
  "io_config": {
    "validation": {
      "use_dagger": false
    }
  }
}
```

### Automatic Fallback

If Dagger fails (Docker not running), go-mcp automatically falls back to local sandbox with a warning.
```

#### 1.3 Create DAGGER.md

Create dedicated Dagger documentation: `docs/DAGGER.md`

```markdown
# Dagger Sandbox Implementation

## Overview

The Dagger sandbox provides containerized execution for data app validation, ensuring consistent and isolated environments.

## Architecture

```
┌─────────────────────────────────────┐
│  IO Provider (validate_data_app)    │
└───────────────┬─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│  Dagger Sandbox                     │
│  ┌─────────────────────────────┐   │
│  │  node:20-alpine3.22         │   │
│  │  ┌─────────────────────┐    │   │
│  │  │  /workspace/        │    │   │
│  │  │  ├── client/        │    │   │
│  │  │  ├── server/        │    │   │
│  │  │  ├── package.json   │    │   │
│  │  │  └── ...            │    │   │
│  │  └─────────────────────┘    │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## Workflow

1. **Scaffold** - Create project on host
2. **RefreshFromHost** - Copy files to container
3. **WithEnvs** - Inject credentials
4. **Exec** - Run validation commands
5. **ExportDirectory** - Extract artifacts (optional)
6. **Close** - Cleanup container

## Performance

### Benchmarks

```
BenchmarkDagger_WriteFiles-8     50    24.5 ms/op
BenchmarkLocal_WriteFiles-8     100    12.1 ms/op
```

**Overhead**: ~2x vs local (acceptable for security/isolation)

### Optimization Tips

1. **Use WriteFiles() for bulk**: Single operation vs N operations
2. **Cache Docker images**: Pre-pull node:20-alpine3.22
3. **Reuse containers**: Fork() instead of recreating
4. **Parallel validation**: Run multiple validations with Fork()

## Troubleshooting

### Docker Not Running

**Symptom**: `failed to connect to Dagger: connection refused`

**Solution**:
```bash
# Start Docker daemon
systemctl start docker  # Linux
open -a Docker          # macOS
```

### Image Pull Slow

**Symptom**: First validation takes minutes

**Solution**:
```bash
# Pre-pull image
docker pull node:20-alpine3.22
```

### Permission Errors

**Symptom**: `permission denied` in container

**Solution**: Check file permissions in scaffolded project

### Network Issues

**Symptom**: `npm install` fails in container

**Solution**: Verify Docker network allows internet access

## Advanced Usage

### Custom Docker Image

```json
{
  "validation": {
    "docker_image": "node:18-bullseye"
  }
}
```

### Custom Timeout

```json
{
  "validation": {
    "timeout": 1200
  }
}
```

### Disable Dagger

```json
{
  "validation": {
    "use_dagger": false
  }
}
```

## Security Considerations

1. **Credential Propagation**: Only DATABRICKS_* env vars propagated
2. **Filesystem Isolation**: Container can't access host outside project
3. **Network Access**: Container needs network for npm install
4. **Resource Limits**: Dagger respects Docker resource constraints
5. **Image Trust**: Use official node images from Docker Hub

## Implementation Details

See `pkg/sandbox/dagger/dagger.go` for full implementation.

**Reference**: Rust implementation at `../agent/edda/edda_sandbox/src/dagger.rs`
```

### Step 2: Performance Optimization

#### 2.1 Connection Pooling

Add singleton Dagger client:

```go
var (
    globalClient *dagger.Client
    clientMu     sync.Mutex
)

func GetGlobalClient(ctx context.Context) (*dagger.Client, error) {
    clientMu.Lock()
    defer clientMu.Unlock()

    if globalClient == nil {
        var err error
        globalClient, err = dagger.Connect(ctx)
        if err != nil {
            return nil, err
        }
    }

    return globalClient, nil
}
```

### Step 4: Final Verification

#### 4.1 Verification Checklist

Run through complete checklist:

```bash
# 1. All operations implemented
go test ./pkg/sandbox/dagger/ -v

# 2. Integration tests pass
go test ./test/integration/ -v -run Dagger

# 3. Performance acceptable
go test -bench=. ./test/integration/

# 4. Race detector clean
go test -race ./pkg/sandbox/dagger/

# 5. Build successful
make build

# 6. Documentation complete
# - CLAUDE.md updated
# - README.md updated
# - DAGGER.md created

# 7. Configuration validated
./go-mcp check
```

#### 4.2 Success Metrics

Define success criteria:

- [ ] All 11 sandbox operations implemented
- [ ] Unit tests >90% coverage
- [ ] Integration tests passing
- [ ] Zero resource leaks
- [ ] Documentation complete
- [ ] Code reviewed and approved

### Step 5: Monitoring & Metrics

Add telemetry to track usage:

```go
type DaggerMetrics struct {
    ValidationCount   int64
    SuccessCount      int64
    FallbackCount     int64
    AverageDurationMs float64
}

func (d *DaggerSandbox) RecordValidation(success bool, duration time.Duration) {
    // Log metrics for monitoring
    d.logger.Info("Validation completed",
        "success", success,
        "duration_ms", duration.Milliseconds(),
        "sandbox", "dagger")
}
```

## Verification Checklist

- [ ] CLAUDE.md updated with Dagger section
- [ ] README.md updated with sandbox modes
- [ ] DAGGER.md created with full documentation
- [ ] Performance optimizations implemented
- [ ] All tests passing
- [ ] Metrics/monitoring added
- [ ] Code reviewed
- [ ] Feature flag implemented

## Success Criteria

- [ ] Complete documentation (CLAUDE.md, README.md, DAGGER.md)
- [ ] Rollout strategy approved
- [ ] All verification checks pass
- [ ] Zero critical issues

## Deliverables

1. **Documentation**:
   - Updated CLAUDE.md
   - Updated README.md
   - New DAGGER.md

2. **Performance**:
   - Benchmark results
   - Optimization recommendations

3. **Verification**:
   - All tests passing
   - Coverage reports
   - Review approval

## Next Steps

After completing this phase:
- **parity-2** is complete (all dependencies resolved)
- Proceed to **parity-3**: Phase 2: Trajectory Tracking (if not already done)
- Or **parity-4**: Phase 3: Developer Experience Improvements
