# Phase 3: Developer Experience Improvements

## Objective

Enhance developer experience with better debugging tools, automatic update notifications, and improved configuration validation. These quality-of-life features improve maintainability and user satisfaction.

## Priority

**🟡 MEDIUM** - High value for users but not blocking core functionality.

## Context

The Rust implementation includes several developer experience features:
- `edda_mcp/src/yell.rs` - Bug reporting with diagnostic bundle
- `edda_mcp/src/version_check.rs` - Auto-update notifications from GitHub
- Enhanced logging and configuration validation

These features make the tool easier to debug, maintain, and keep up-to-date.

## Implementation Steps

### Feature 1: Bug Reporting Command

#### Objective
Create a CLI command that collects diagnostic information into a tarball for easy bug reporting.

#### Reference
`/Users/fabian.jakobs/Workspaces/agent/edda/edda_mcp/src/yell.rs`

#### Implementation

##### 1.1 Create Report Package (`pkg/report/`)

**report.go:**

```go
package report

import (
    "archive/tar"
    "compress/gzip"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "runtime"
    "time"

    "go-mcp/pkg/version"
)

// Metadata contains system and version information
type Metadata struct {
    Timestamp   time.Time `json:"timestamp"`
    OS          string    `json:"os"`
    Arch        string    `json:"arch"`
    Version     string    `json:"version"`
    BinaryHash  string    `json:"binary_sha256"`
    Description string    `json:"description"`
}

// BugReport collects diagnostic information
type BugReport struct {
    description string
    metadata    Metadata
}

// NewBugReport creates a new bug report
func NewBugReport(description string) (*BugReport, error) {
    binaryHash, err := computeBinaryHash()
    if err != nil {
        return nil, fmt.Errorf("failed to compute binary hash: %w", err)
    }

    return &BugReport{
        description: description,
        metadata: Metadata{
            Timestamp:   time.Now(),
            OS:          runtime.GOOS,
            Arch:        runtime.GOARCH,
            Version:     version.Version,
            BinaryHash:  binaryHash,
            Description: description,
        },
    }, nil
}

// computeBinaryHash computes SHA256 of the running binary
func computeBinaryHash() (string, error) {
    executable, err := os.Executable()
    if err != nil {
        return "", err
    }

    file, err := os.Open(executable)
    if err != nil {
        return "", err
    }
    defer file.Close()

    hasher := sha256.New()
    if _, err := io.Copy(hasher, file); err != nil {
        return "", err
    }

    return hex.EncodeToString(hasher.Sum(nil)), nil
}

// Generate creates the bug report tarball
// Reference: edda_mcp/src/yell.rs:28-115
func (br *BugReport) Generate() (string, error) {
    homeDir, err := os.UserHomeDir()
    if err != nil {
        return "", err
    }

    tmpDir := os.TempDir()
    timestamp := time.Now().Format("20060102-150405")
    filename := filepath.Join(tmpDir, fmt.Sprintf("go-mcp-bug-report-%s.tar.gz", timestamp))

    // Create tarball
    file, err := os.Create(filename)
    if err != nil {
        return "", fmt.Errorf("failed to create archive: %w", err)
    }
    defer file.Close()

    gzWriter := gzip.NewWriter(file)
    defer gzWriter.Close()

    tarWriter := tar.NewWriter(gzWriter)
    defer tarWriter.Close()

    // Add description
    if err := br.addDescription(tarWriter); err != nil {
        return "", err
    }

    // Add metadata
    if err := br.addMetadata(tarWriter); err != nil {
        return "", err
    }

    // Add trajectory history
    historyFile := filepath.Join(homeDir, ".go-mcp", "history.jsonl")
    if err := br.addFile(tarWriter, historyFile, "history.jsonl"); err != nil {
        // Non-fatal, trajectory might not be enabled
        fmt.Fprintf(os.Stderr, "warning: failed to add history: %v\n", err)
    }

    // Add logs from last 12 hours
    if err := br.addRecentLogs(tarWriter, homeDir); err != nil {
        fmt.Fprintf(os.Stderr, "warning: failed to add logs: %v\n", err)
    }

    return filename, nil
}

// addDescription adds the bug description to the archive
func (br *BugReport) addDescription(tw *tar.Writer) error {
    content := []byte(br.description)

    header := &tar.Header{
        Name: "description.txt",
        Mode: 0644,
        Size: int64(len(content)),
    }

    if err := tw.WriteHeader(header); err != nil {
        return err
    }

    _, err := tw.Write(content)
    return err
}

// addMetadata adds metadata JSON to the archive
func (br *BugReport) addMetadata(tw *tar.Writer) error {
    content, err := json.MarshalIndent(br.metadata, "", "  ")
    if err != nil {
        return err
    }

    header := &tar.Header{
        Name: "metadata.json",
        Mode: 0644,
        Size: int64(len(content)),
    }

    if err := tw.WriteHeader(header); err != nil {
        return err
    }

    _, err = tw.Write(content)
    return err
}

// addFile adds a file to the archive
func (br *BugReport) addFile(tw *tar.Writer, srcPath, destPath string) error {
    file, err := os.Open(srcPath)
    if err != nil {
        return err
    }
    defer file.Close()

    stat, err := file.Stat()
    if err != nil {
        return err
    }

    header := &tar.Header{
        Name: destPath,
        Mode: 0644,
        Size: stat.Size(),
    }

    if err := tw.WriteHeader(header); err != nil {
        return err
    }

    _, err = io.Copy(tw, file)
    return err
}

// addRecentLogs adds logs from the last 12 hours
// Reference: edda_mcp/src/yell.rs:72-98
func (br *BugReport) addRecentLogs(tw *tar.Writer, homeDir string) error {
    logsDir := filepath.Join(homeDir, ".go-mcp", "logs")

    entries, err := os.ReadDir(logsDir)
    if err != nil {
        return err
    }

    cutoff := time.Now().Add(-12 * time.Hour)

    for _, entry := range entries {
        if entry.IsDir() {
            continue
        }

        info, err := entry.Info()
        if err != nil {
            continue
        }

        if info.ModTime().Before(cutoff) {
            continue
        }

        srcPath := filepath.Join(logsDir, entry.Name())
        destPath := filepath.Join("logs", entry.Name())

        if err := br.addFile(tw, srcPath, destPath); err != nil {
            fmt.Fprintf(os.Stderr, "warning: failed to add log %s: %v\n", entry.Name(), err)
        }
    }

    return nil
}
```

##### 1.2 Add CLI Command

**cmd/go-mcp/cli.go:**

```go
var reportBugCmd = &cobra.Command{
    Use:   "report-bug [description]",
    Short: "Generate bug report with diagnostic information",
    Long: `Collects diagnostic information into a tarball for bug reporting.

Includes:
- Bug description
- System metadata (OS, arch, version, binary hash)
- Trajectory history (if available)
- Logs from the last 12 hours

The generated tarball can be shared with maintainers for debugging.`,
    Args: cobra.MaximumNArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        var description string

        if len(args) > 0 {
            description = args[0]
        } else {
            // Prompt for description
            fmt.Print("Please describe the bug:\n> ")
            reader := bufio.NewReader(os.Stdin)
            desc, err := reader.ReadString('\n')
            if err != nil {
                return fmt.Errorf("failed to read description: %w", err)
            }
            description = strings.TrimSpace(desc)
        }

        if description == "" {
            return fmt.Errorf("bug description is required")
        }

        bugReport, err := report.NewBugReport(description)
        if err != nil {
            return fmt.Errorf("failed to create bug report: %w", err)
        }

        fmt.Println("Collecting diagnostic information...")
        filename, err := bugReport.Generate()
        if err != nil {
            return fmt.Errorf("failed to generate report: %w", err)
        }

        fmt.Printf("\nBug report created: %s\n", filename)
        fmt.Println("\nPlease share this file with the maintainers.")
        fmt.Println("You can attach it to a GitHub issue or send via email.")

        return nil
    },
}

func init() {
    rootCmd.AddCommand(reportBugCmd)
}
```

#### Testing

```go
func TestBugReport_Generate(t *testing.T) {
    report, err := NewBugReport("Test bug description")
    if err != nil {
        t.Fatalf("failed to create report: %v", err)
    }

    filename, err := report.Generate()
    if err != nil {
        t.Fatalf("failed to generate report: %v", err)
    }

    // Verify file exists
    if _, err := os.Stat(filename); err != nil {
        t.Errorf("report file not found: %v", err)
    }

    // Clean up
    os.Remove(filename)
}
```

### Feature 2: Version Check and Auto-Update Notification

#### Objective
Check GitHub releases for newer versions and notify users.

#### Reference
`/Users/fabian.jakobs/Workspaces/agent/edda/edda_mcp/src/version_check.rs`

#### Implementation

##### 2.1 Create Version Check Package (`pkg/versioncheck/`)

**versioncheck.go:**

```go
package versioncheck

import (
    "context"
    "encoding/json"
    "fmt"
    "net/http"
    "time"

    "github.com/Masterminds/semver/v3"
    "go-mcp/pkg/version"
)

const (
    githubReleasesURL = "https://api.github.com/repos/appdotbuild/agent/releases/latest"
    checkTimeout      = 5 * time.Second
)

// Release represents a GitHub release
type Release struct {
    TagName string `json:"tag_name"`
    Name    string `json:"name"`
    HTMLURL string `json:"html_url"`
}

// CheckForUpdate checks if a newer version is available
// Reference: edda_mcp/src/version_check.rs:15-70
func CheckForUpdate(ctx context.Context) {
    // Run in background goroutine
    go func() {
        ctx, cancel := context.WithTimeout(ctx, checkTimeout)
        defer cancel()

        result := checkLatestVersion(ctx)
        if result != "" {
            fmt.Println(result)
        }
    }()
}

// checkLatestVersion performs the actual check
func checkLatestVersion(ctx context.Context) string {
    req, err := http.NewRequestWithContext(ctx, "GET", githubReleasesURL, nil)
    if err != nil {
        return ""
    }

    req.Header.Set("User-Agent", "go-mcp")

    client := &http.Client{Timeout: checkTimeout}
    resp, err := client.Do(req)
    if err != nil {
        return ""
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        return ""
    }

    var release Release
    if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
        return ""
    }

    return compareVersions(release)
}

// compareVersions compares current version with latest release
func compareVersions(release Release) string {
    currentVer, err := semver.NewVersion(version.Version)
    if err != nil {
        return ""
    }

    latestVer, err := semver.NewVersion(release.TagName)
    if err != nil {
        return ""
    }

    if latestVer.GreaterThan(currentVer) {
        return fmt.Sprintf(`
╔═══════════════════════════════════════════════════════════╗
║                   UPDATE AVAILABLE                        ║
║                                                           ║
║  Current version: %s                                    ║
║  Latest version:  %s                                    ║
║                                                           ║
║  To upgrade, run:                                        ║
║    go install go-mcp@latest                              ║
║                                                           ║
║  Release notes:                                          ║
║    %s                                                     ║
╚═══════════════════════════════════════════════════════════╝
`,
            currentVer.String(),
            latestVer.String(),
            release.HTMLURL,
        )
    }

    return ""
}
```

##### 2.2 Integrate with Server Start

**cmd/go-mcp/cli.go:**

```go
var startCmd = &cobra.Command{
    Use:   "start",
    Short: "Start the MCP server",
    RunE: func(cmd *cobra.Command, args []string) error {
        // Check for updates (non-blocking)
        versioncheck.CheckForUpdate(context.Background())

        // Start server
        // ... existing code ...
    },
}
```

#### Testing

```go
func TestCheckForUpdate_NonBlocking(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
    defer cancel()

    // Should return immediately, not block
    start := time.Now()
    CheckForUpdate(ctx)
    duration := time.Since(start)

    if duration > 100*time.Millisecond {
        t.Errorf("CheckForUpdate blocked for %v, expected immediate return", duration)
    }
}

func TestCompareVersions(t *testing.T) {
    tests := []struct {
        name    string
        current string
        latest  string
        expect  bool // true if update notification expected
    }{
        {"same version", "1.0.0", "1.0.0", false},
        {"newer available", "1.0.0", "1.1.0", true},
        {"older release", "2.0.0", "1.0.0", false},
        {"patch update", "1.0.0", "1.0.1", true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test version comparison logic
        })
    }
}
```

### Feature 3: Enhanced Configuration Validation

#### Objective
Improve the `go-mcp check` command with comprehensive validation.

#### Implementation

**cmd/go-mcp/cli.go:**

```go
var checkCmd = &cobra.Command{
    Use:   "check",
    Short: "Validate environment and configuration",
    Long: `Performs comprehensive environment checks:
- Configuration file validation
- Environment variables
- Docker availability (for Dagger sandbox)
- Databricks connectivity
- SQL warehouse access`,
    RunE: func(cmd *cobra.Command, args []string) error {
        fmt.Println("🔍 Checking go-mcp environment...\n")

        allOK := true

        // 1. Configuration
        fmt.Print("Configuration file... ")
        cfg, err := config.Load(cfgFile)
        if err != nil {
            fmt.Printf("❌ FAILED: %v\n", err)
            allOK = false
        } else {
            fmt.Println("✅ OK")
        }

        // 2. Environment variables
        fmt.Print("Environment variables... ")
        requiredEnvVars := []string{"DATABRICKS_HOST", "DATABRICKS_TOKEN"}
        missingVars := []string{}
        for _, v := range requiredEnvVars {
            if os.Getenv(v) == "" {
                missingVars = append(missingVars, v)
            }
        }
        if len(missingVars) > 0 {
            fmt.Printf("❌ MISSING: %s\n", strings.Join(missingVars, ", "))
            allOK = false
        } else {
            fmt.Println("✅ OK")
        }

        // 3. Docker
        fmt.Print("Docker daemon... ")
        cmd := exec.Command("docker", "info")
        if err := cmd.Run(); err != nil {
            fmt.Printf("❌ NOT AVAILABLE: %v\n", err)
            fmt.Println("   Note: Dagger sandbox requires Docker")
            allOK = false
        } else {
            fmt.Println("✅ OK")
        }

        // 4. Databricks connectivity
        if cfg != nil {
            fmt.Print("Databricks connectivity... ")
            // Try to list catalogs
            if err := testDatabricksConnection(cfg); err != nil {
                fmt.Printf("❌ FAILED: %v\n", err)
                allOK = false
            } else {
                fmt.Println("✅ OK")
            }

            // 5. SQL warehouse
            if cfg.WarehouseID != "" {
                fmt.Print("SQL warehouse access... ")
                if err := testWarehouseAccess(cfg); err != nil {
                    fmt.Printf("❌ FAILED: %v\n", err)
                    allOK = false
                } else {
                    fmt.Println("✅ OK")
                }
            }
        }

        fmt.Println()
        if allOK {
            fmt.Println("✅ All checks passed!")
            return nil
        } else {
            fmt.Println("❌ Some checks failed. Please fix the issues above.")
            return fmt.Errorf("environment check failed")
        }
    },
}
```

### Feature 4: Enhanced Logging

#### Objective
Improve logging with filtering, sampling, and performance metrics.

#### Implementation

**pkg/logging/logging.go:**

```go
// Add log level filtering
func NewLogger(sessionID string, level slog.Level) (*slog.Logger, error) {
    // ... existing code ...

    // Add level filter
    handler = &LevelFilter{
        Handler: handler,
        Level:   level,
    }

    return slog.New(handler), nil
}

// LevelFilter filters logs by level
type LevelFilter struct {
    Handler slog.Handler
    Level   slog.Level
}

func (f *LevelFilter) Enabled(ctx context.Context, level slog.Level) bool {
    return level >= f.Level
}

func (f *LevelFilter) Handle(ctx context.Context, record slog.Record) error {
    if record.Level >= f.Level {
        return f.Handler.Handle(ctx, record)
    }
    return nil
}

// Add performance logging helper
func LogDuration(logger *slog.Logger, operation string, start time.Time) {
    duration := time.Since(start)
    logger.Info("operation completed",
        "operation", operation,
        "duration_ms", duration.Milliseconds(),
    )
}
```

## Success Criteria

- [ ] Bug reporting command working
  - [ ] Collects description, metadata, history, logs
  - [ ] Creates tarball successfully
  - [ ] File can be easily shared
- [ ] Version check implemented
  - [ ] Non-blocking background check
  - [ ] Compares semantic versions correctly
  - [ ] Displays upgrade instructions
- [ ] Enhanced `check` command
  - [ ] Validates all configuration
  - [ ] Tests Docker availability
  - [ ] Tests Databricks connectivity
  - [ ] Tests warehouse access
- [ ] Enhanced logging
  - [ ] Level filtering working
  - [ ] Performance metrics logged
- [ ] All tests passing
- [ ] Documentation updated

## Dependencies

- `github.com/Masterminds/semver/v3` - Semantic version comparison
- Standard library for tar/gzip

## Timeline

- **Days 1-2**: Bug reporting command
- **Days 3-4**: Version check
- **Day 5**: Enhanced check command
- **Day 6**: Enhanced logging
- **Day 7**: Testing, documentation, polish

## Documentation Updates

Add to CLAUDE.md:

```markdown
## Developer Tools

### Bug Reporting
```bash
go-mcp report-bug "Description of the issue"
```
Collects diagnostic bundle with history, logs, and metadata.

### Version Check
Automatically checks for updates on startup (non-blocking).
Manual check: `go-mcp version`

### Environment Validation
```bash
go-mcp check
```
Validates configuration, Docker, Databricks connectivity.
```

## Next Phase

After completing developer experience improvements, proceed to:
- **Phase 4**: Architecture Improvements
