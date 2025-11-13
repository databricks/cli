# Dagger Phase 3: Integration, Testing & Configuration

**Issue**: parity-46
**Parent**: parity-2 (Phase 1: Dagger Sandbox Implementation)
**Depends On**: parity-44, parity-45
**Estimated Time**: 1 week

## Objective

Integrate Dagger sandbox with IO Provider validation, write comprehensive integration tests, and update configuration system to support Dagger/local fallback.

## Prerequisites

- Phase 1 (parity-44) and Phase 2 (parity-45) completed
- All core and advanced operations working
- Docker daemon running

## Implementation Steps

### Step 1: Configuration Updates

#### 1.1 Update ValidationConfig

Modify `pkg/config/config.go`:

```go
type ValidationConfig struct {
    Command     string `json:"command,omitempty"`
    DockerImage string `json:"docker_image,omitempty"`
    UseDagger   bool   `json:"use_dagger"`
    Timeout     int    `json:"timeout,omitempty"`
}

func (c *ValidationConfig) SetDefaults() {
    if c.DockerImage == "" {
        c.DockerImage = "node:20-alpine3.22"
    }
    if c.Timeout == 0 {
        c.Timeout = 600
    }
    if !c.hasExplicitUseDagger() {
        c.UseDagger = true
    }
}

func (c *ValidationConfig) hasExplicitUseDagger() bool {
    return true
}
```

#### 1.2 Example Config

Update `~/.go-mcp/config.json` documentation:

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

### Step 2: IO Provider Integration

#### 2.1 Update validate.go

Modify `pkg/providers/io/validate.go`:

```go
func (p *Provider) validateDataApp(ctx context.Context, args ValidateDataAppInput) (*ValidateDataAppResult, error) {
    validationCfg := p.config.IoConfig.Validation

    var sb sandbox.Sandbox
    var err error

    if validationCfg.UseDagger {
        sb, err = p.createDaggerSandbox(ctx, args.WorkDir, validationCfg)
        if err != nil {
            p.logger.Warn("Failed to create Dagger sandbox, falling back to local",
                "error", err.Error())
            sb, err = p.createLocalSandbox(args.WorkDir)
        }
    } else {
        sb, err = p.createLocalSandbox(args.WorkDir)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to create sandbox: %w", err)
    }
    defer sb.Close()

    return p.runValidation(ctx, sb, args)
}

func (p *Provider) createDaggerSandbox(ctx context.Context, workDir string, cfg *config.ValidationConfig) (sandbox.Sandbox, error) {
    sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
        Image:          cfg.DockerImage,
        ExecuteTimeout: cfg.Timeout,
        BaseDir:        workDir,
    })
    if err != nil {
        return nil, err
    }

    if err := p.propagateEnvironment(sb); err != nil {
        sb.Close()
        return nil, fmt.Errorf("failed to set environment: %w", err)
    }

    if err := sb.RefreshFromHost(ctx, workDir, "/workspace"); err != nil {
        sb.Close()
        return nil, fmt.Errorf("failed to sync files: %w", err)
    }

    return sb, nil
}

func (p *Provider) createLocalSandbox(workDir string) (sandbox.Sandbox, error) {
    return local.NewLocalSandbox(workDir)
}

func (p *Provider) propagateEnvironment(sb sandbox.Sandbox) error {
    daggerSb, ok := sb.(*dagger.DaggerSandbox)
    if !ok {
        return nil
    }

    envVars := []string{
        "DATABRICKS_HOST",
        "DATABRICKS_TOKEN",
        "DATABRICKS_WAREHOUSE_ID",
    }

    for _, key := range envVars {
        if value := os.Getenv(key); value != "" {
            daggerSb.WithEnv(key, value)
        }
    }

    return nil
}
```

### Step 3: Comprehensive Integration Tests

#### 3.1 Dagger Validation Workflow Test

Create `test/integration/dagger_validation_test.go`:

```go
package integration

import (
    "context"
    "os"
    "path/filepath"
    "testing"

    "go-mcp/pkg/config"
    "go-mcp/pkg/providers/io"
    "go-mcp/pkg/sandbox/dagger"
    "go-mcp/pkg/session"
)

func TestDaggerValidation_TRPCProject(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    if !isDaggerAvailable() {
        t.Skip("Docker not available")
    }

    ctx := context.Background()
    tmpDir := t.TempDir()

    cfg := &config.Config{
        IoConfig: &config.IoConfig{
            Validation: &config.ValidationConfig{
                UseDagger:   true,
                DockerImage: "node:20-alpine3.22",
                Timeout:     600,
            },
        },
    }

    sess := session.NewSession()
    provider := io.NewProvider(cfg, sess, logger)

    scaffoldResult, err := provider.ScaffoldDataApp(ctx, io.ScaffoldDataAppInput{
        AppName:       "test-app",
        AppDirectory:  tmpDir,
        TemplateName:  "trpc_starter",
    })
    if err != nil {
        t.Fatalf("scaffold failed: %v", err)
    }

    validateResult, err := provider.ValidateDataApp(ctx, io.ValidateDataAppInput{
        WorkDir: scaffoldResult.ProjectPath,
    })
    if err != nil {
        t.Fatalf("validation failed: %v", err)
    }

    if !validateResult.Success {
        t.Errorf("validation should succeed, got: %s", validateResult.Message)
    }

    if validateResult.SandboxType != "dagger" {
        t.Errorf("expected dagger sandbox, got: %s", validateResult.SandboxType)
    }
}

func TestDaggerValidation_WithDatabricksEnv(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    if !isDaggerAvailable() {
        t.Skip("Docker not available")
    }

    os.Setenv("DATABRICKS_HOST", "https://test.databricks.com")
    os.Setenv("DATABRICKS_TOKEN", "test-token")
    defer os.Unsetenv("DATABRICKS_HOST")
    defer os.Unsetenv("DATABRICKS_TOKEN")

    ctx := context.Background()

    sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{})
    if err != nil {
        t.Fatalf("failed to create sandbox: %v", err)
    }
    defer sb.Close()

    sb.WithEnv("DATABRICKS_HOST", os.Getenv("DATABRICKS_HOST"))
    sb.WithEnv("DATABRICKS_TOKEN", os.Getenv("DATABRICKS_TOKEN"))

    result, err := sb.Exec(ctx, "echo $DATABRICKS_HOST")
    if err != nil {
        t.Fatalf("exec failed: %v", err)
    }

    if !strings.Contains(result.Stdout, "test.databricks.com") {
        t.Errorf("environment not propagated correctly")
    }
}

func TestLocalFallback_WhenDaggerUnavailable(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test")
    }

    ctx := context.Background()
    tmpDir := t.TempDir()

    cfg := &config.Config{
        IoConfig: &config.IoConfig{
            Validation: &config.ValidationConfig{
                UseDagger: true,
            },
        },
    }

    sess := session.NewSession()
    provider := io.NewProvider(cfg, sess, logger)

    scaffoldResult, err := provider.ScaffoldDataApp(ctx, io.ScaffoldDataAppInput{
        AppName:       "test-app",
        AppDirectory:  tmpDir,
        TemplateName:  "trpc_starter",
    })
    if err != nil {
        t.Fatalf("scaffold failed: %v", err)
    }

    validateResult, err := provider.ValidateDataApp(ctx, io.ValidateDataAppInput{
        WorkDir: scaffoldResult.ProjectPath,
    })
    if err != nil {
        t.Fatalf("validation failed: %v", err)
    }

    if !validateResult.Success {
        t.Errorf("validation should succeed with local fallback")
    }
}

func isDaggerAvailable() bool {
    ctx := context.Background()
    sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{})
    if err != nil {
        return false
    }
    sb.Close()
    return true
}
```

#### 3.2 Performance Comparison Test

Create `test/integration/sandbox_benchmark_test.go`:

```go
package integration

import (
    "context"
    "testing"

    "go-mcp/pkg/sandbox/dagger"
    "go-mcp/pkg/sandbox/local"
)

func BenchmarkDagger_WriteFiles(b *testing.B) {
    if testing.Short() {
        b.Skip("skipping benchmark")
    }

    ctx := context.Background()
    sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{})
    if err != nil {
        b.Fatalf("failed to create sandbox: %v", err)
    }
    defer sb.Close()

    files := generateTestFiles(100)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if err := sb.WriteFiles(ctx, files); err != nil {
            b.Fatalf("write failed: %v", err)
        }
    }
}

func BenchmarkLocal_WriteFiles(b *testing.B) {
    if testing.Short() {
        b.Skip("skipping benchmark")
    }

    tmpDir := b.TempDir()
    sb, err := local.NewLocalSandbox(tmpDir)
    if err != nil {
        b.Fatalf("failed to create sandbox: %v", err)
    }

    files := generateTestFiles(100)

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        if err := sb.WriteFiles(context.Background(), files); err != nil {
            b.Fatalf("write failed: %v", err)
        }
    }
}

func generateTestFiles(count int) map[string]string {
    files := make(map[string]string)
    for i := 0; i < count; i++ {
        files[fmt.Sprintf("file%d.txt", i)] = fmt.Sprintf("content %d", i)
    }
    return files
}
```

### Step 4: Error Handling & Logging

Add comprehensive logging to track Dagger operations:

```go
func (p *Provider) createDaggerSandbox(ctx context.Context, workDir string, cfg *config.ValidationConfig) (sandbox.Sandbox, error) {
    p.logger.Info("Creating Dagger sandbox",
        "image", cfg.DockerImage,
        "timeout", cfg.Timeout,
        "workDir", workDir)

    sb, err := dagger.NewDaggerSandbox(ctx, dagger.Config{
        Image:          cfg.DockerImage,
        ExecuteTimeout: cfg.Timeout,
        BaseDir:        workDir,
    })
    if err != nil {
        p.logger.Error("Failed to create Dagger sandbox",
            "error", err.Error(),
            "image", cfg.DockerImage)
        return nil, err
    }

    p.logger.Debug("Propagating environment variables")
    if err := p.propagateEnvironment(sb); err != nil {
        p.logger.Error("Failed to propagate environment", "error", err.Error())
        sb.Close()
        return nil, err
    }

    p.logger.Debug("Syncing files from host to container", "workDir", workDir)
    if err := sb.RefreshFromHost(ctx, workDir, "/workspace"); err != nil {
        p.logger.Error("Failed to sync files", "error", err.Error())
        sb.Close()
        return nil, err
    }

    p.logger.Info("Dagger sandbox created successfully")
    return sb, nil
}
```

## Verification Checklist

- [ ] ValidationConfig updated with Dagger options
- [ ] IO Provider uses Dagger by default
- [ ] Fallback to local sandbox works
- [ ] Environment variables propagated correctly
- [ ] Integration tests pass
- [ ] Performance benchmarks run
- [ ] Logging tracks all operations
- [ ] Error handling is comprehensive
- [ ] Config file documentation updated

## Success Criteria

- [ ] Validation uses Dagger when configured
- [ ] Graceful fallback when Docker unavailable
- [ ] Integration tests pass (scaffold → validate workflow)
- [ ] Performance overhead <100% vs local
- [ ] All environment variables propagated
- [ ] Clear logging for debugging
- [ ] Configuration well-documented

## Known Issues

1. **First Run Slow**: Initial image pull can take time
2. **Network Required**: npm install needs network access
3. **Platform Differences**: ARM vs x86 images may behave differently

## Next Steps

After completing this phase, proceed to:
- **parity-47**: Documentation, Performance & Rollout
