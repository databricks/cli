# Phase 3: Integration with MCP Server

## Objective
Integrate the deployment provider into the main MCP server, ensuring it's conditionally enabled based on configuration and properly wired into the server startup.

## Context
The deployment provider should only be enabled when `config.AllowDeployment == true`. This follows the security model where deployment is an opt-in feature.

Reference: `cmd/go-mcp/cli.go` and `edda_mcp/src/providers/mod.rs:95-107`

## Prerequisites
- Phase 1 completed (`pkg/providers/databricks/deployment.go`)
- Phase 2 completed (`pkg/providers/deployment/provider.go`)

## Implementation Details

### File to Modify
`go-mcp/cmd/go-mcp/cli.go`

### Changes Required

#### 1. Add Import

Add to the import section:
```go
import (
    // ... existing imports ...
    "github.com/appdotbuild/go-mcp/pkg/providers/deployment"
)
```

#### 2. Modify runServer Function

Locate the `runServer` function where providers are registered. Currently it looks like this:

```go
func runServer(cmd *cobra.Command, args []string) error {
    // ... existing code to load config, create session, logger ...

    // Create and register providers
    databricksProvider, err := databricks.NewProvider(cfg, sess, logger)
    // ...
    ioProvider, err := io.NewProvider(cfg.IOConfig, sess, logger)
    // ...

    if cfg.WithWorkspaceTools {
        workspaceProvider, err := workspace.NewProvider(sess, logger)
        // ...
    }

    // Start server
    // ...
}
```

Add the deployment provider registration **after** the workspace provider section:

```go
    // Register deployment provider if enabled
    if cfg.AllowDeployment {
        logger.Info("Deployment provider enabled")
        deploymentProvider, err := deployment.NewProvider(cfg, sess, logger)
        if err != nil {
            return fmt.Errorf("failed to create deployment provider: %w", err)
        }

        if err := deploymentProvider.RegisterTools(server); err != nil {
            return fmt.Errorf("failed to register deployment tools: %w", err)
        }
    } else {
        logger.Info("Deployment provider disabled (enable with allow_deployment: true)")
    }
```

### Configuration Verification

Ensure the config structure in `pkg/config/config.go` has the necessary fields (should already exist from analysis):

```go
type Config struct {
    AllowDeployment    bool     `json:"allow_deployment" mapstructure:"allow_deployment"`
    // ... other fields ...
}
```

And ensure environment variable loading includes warehouse ID (should already exist):

```go
func (c *Config) LoadFromEnv() error {
    // ... existing code ...

    if warehouseID := os.Getenv("DATABRICKS_WAREHOUSE_ID"); warehouseID != "" {
        c.WarehouseID = warehouseID
    }

    return nil
}
```

### Example Configuration File

Users should have a config like this in `~/.go-mcp/config.json`:

```json
{
  "allow_deployment": true,
  "with_workspace_tools": true,
  "required_providers": ["databricks", "io"],
  "warehouse_id": "your-warehouse-id",
  "io_config": {
    "template": {
      "name": "default"
    }
  }
}
```

### Environment Variables

Ensure documentation mentions these required env vars:
```bash
DATABRICKS_HOST=https://your-workspace.databricks.com
DATABRICKS_TOKEN=dapi...
DATABRICKS_WAREHOUSE_ID=your-warehouse-id  # Required for deployment
```

## Testing the Integration

### Manual Test

1. Build the binary:
```bash
cd go-mcp
make build
```

2. Create test config with deployment enabled:
```bash
cat > ~/.go-mcp/config.json << 'EOF'
{
  "allow_deployment": true,
  "with_workspace_tools": true,
  "required_providers": ["databricks", "io"]
}
EOF
```

3. Set environment variables:
```bash
export DATABRICKS_HOST=https://your-workspace.databricks.com
export DATABRICKS_TOKEN=dapi...
export DATABRICKS_WAREHOUSE_ID=your-warehouse-id
```

4. Run the server:
```bash
./go-mcp --config ~/.go-mcp/config.json
```

5. Verify in logs:
- Should see: "Deployment provider enabled"
- Should see: "Registering deployment tools"

### Test with MCP Client

Create a simple test script to verify the tool is registered:

```bash
# Send initialize request and check tool list
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"test","version":"1.0"}}}' | ./go-mcp
```

Look for `deploy_databricks_app` in the tools list.

## Verification Checklist

After implementation:
- [ ] Import statement added for deployment provider
- [ ] Deployment provider created and registered conditionally
- [ ] Logs indicate whether deployment is enabled/disabled
- [ ] Binary builds without errors: `make build`
- [ ] Server starts successfully with deployment enabled
- [ ] Server starts successfully with deployment disabled
- [ ] Tool appears in tools list when enabled
- [ ] Tool does not appear when disabled

## Success Criteria
- [ ] Deployment provider integrated into CLI
- [ ] Conditional registration based on config.AllowDeployment
- [ ] Appropriate logging for enabled/disabled state
- [ ] No compilation errors
- [ ] Server starts successfully in both modes
