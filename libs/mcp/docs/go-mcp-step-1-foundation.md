# Step 1: Project Setup and Foundation

## Overview
Set up the Go project structure, initialize dependencies, and create foundational utilities for configuration, logging, and error handling.

## Tasks

### 1.1 Initialize Go Module

```bash
mkdir go-mcp
cd go-mcp
go mod init github.com/yourusername/go-mcp
```

Create the directory structure:
```
go-mcp/
├── cmd/
│   └── go-mcp/
│       └── main.go
├── pkg/
│   ├── config/
│   │   ├── config.go
│   │   └── config_test.go
│   ├── errors/
│   │   └── errors.go
│   └── version/
│       └── version.go
├── internal/
│   └── templates/
│       └── embed.go
├── go.mod
├── go.sum
└── README.md
```

### 1.2 Add Core Dependencies

```bash
go get github.com/databricks/databricks-sdk-go
go get github.com/spf13/viper
```

### 1.3 Implement Configuration System

**pkg/config/config.go:**

Define configuration structure:
```go
type Config struct {
    AllowDeployment    bool
    WithWorkspaceTools bool
    RequiredProviders  []string
    IOConfig          *IOConfig
    DatabricksHost    string
    DatabricksToken   string
    WarehouseID       string
}

type IOConfig struct {
    Template   TemplateConfig
    Validation *ValidationConfig
}

type TemplateConfig struct {
    Name string
    Path string
}

type ValidationConfig struct {
    Command     string
    DockerImage string
}
```

Functions to implement:
- `LoadConfig() (*Config, error)` - Load from ~/.go-mcp/config.json
- `LoadFromEnv() (*Config, error)` - Override with environment variables
- `Validate() error` - Validate required fields
- `DefaultConfig() *Config` - Return default configuration

**pkg/config/config_test.go:**

Test cases:
- Test loading default config
- Test loading from file
- Test environment variable overrides
- Test validation errors
- Test missing config file (should create with defaults)

### 1.4 Implement Error Handling

**pkg/errors/errors.go:**

Create custom error types:
```go
type MCPError struct {
    Code    int
    Message string
    Details map[string]interface{}
}

func (e *MCPError) Error() string

// Constructor functions
func InvalidParams(message string) *MCPError
func InvalidRequest(message string) *MCPError
func InternalError(message string) *MCPError
func ProviderNotAvailable(provider string) *MCPError
```

### 1.5 Implement Version Management

**pkg/version/version.go:**

```go
var (
    Version   = "dev"
    Commit    = "unknown"
    BuildTime = "unknown"
)

func GetVersion() string
func CheckForUpdates(ctx context.Context) error
```

Version checking:
- Make HTTP request to GitHub releases API
- Compare semantic versions
- Log warning if newer version available
- Non-blocking, timeout after 5 seconds

### 1.6 Create Main Entry Point

**cmd/go-mcp/main.go:**

Skeleton:
```go
func main() {
    cfg, err := config.LoadConfig()
    if err != nil {
        log.Fatalf("Failed to load config: %v", err)
    }

    if err := cfg.LoadFromEnv(); err != nil {
        log.Fatalf("Failed to load env overrides: %v", err)
    }

    if err := cfg.Validate(); err != nil {
        log.Fatalf("Invalid configuration: %v", err)
    }

    // Version check in background
    go version.CheckForUpdates(context.Background())

    // TODO: Initialize and start server
}
```

Add flags:
- `--disallow-deployment` - Disable deployment operations
- `--with-workspace-tools` - Enable workspace tools
- `--config` - Override config file path
- `--version` - Print version and exit

### 1.7 Implement Logging

Set up structured logging:
- Use standard library `log/slog` (Go 1.21+)
- Log levels: DEBUG, INFO, WARN, ERROR
- Session-based log files in ~/.go-mcp/logs/
- Console logging for development
- Log rotation (keep last 10 sessions)

Create logger in `pkg/logging/logging.go`:
```go
func NewLogger(sessionID string, toFile bool) *slog.Logger
func SetLogLevel(level string) error
```

### 1.8 Add Basic Tests

Create `pkg/config/config_test.go` with table-driven tests:
```go
func TestLoadConfig(t *testing.T) {
    tests := []struct {
        name    string
        setup   func() (string, error)  // Create temp config
        want    *Config
        wantErr bool
    }{
        {
            name: "default config",
            // ...
        },
        {
            name: "custom config",
            // ...
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Acceptance Criteria

- [ ] Go module initializes successfully
- [ ] Config loads from file and environment
- [ ] Version info displays correctly
- [ ] Error types provide useful context
- [ ] Logging works to file and console
- [ ] All unit tests pass
- [ ] `go build ./cmd/go-mcp` succeeds
- [ ] `./go-mcp --version` prints version info

## Testing Commands

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./pkg/config/...

# Build binary
go build -o go-mcp ./cmd/go-mcp

# Test binary
./go-mcp --version
./go-mcp --help
```

## Next Steps

Proceed to Step 2: Sandbox Abstraction once all acceptance criteria are met.
