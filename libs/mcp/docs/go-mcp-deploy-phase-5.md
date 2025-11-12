# Phase 5: Documentation

## Objective
Update documentation to reflect the new deployment functionality, providing clear guidance for users and developers.

## Context
Documentation should match the style and format of existing docs in the Go MCP project, following the patterns in `go-mcp/CLAUDE.md` and `go-mcp/README.md`.

## Prerequisites
- All previous phases completed
- Deployment functionality tested and working

## Implementation Details

### File 1: Update CLAUDE.md

File: `go-mcp/CLAUDE.md`

#### Section 1: Add to Architecture Diagram

Update the architecture section (around line 18) to include deployment:

```markdown
```
go-mcp/
├── cmd/go-mcp/          # Entry point and CLI (Cobra)
├── pkg/
│   ├── config/          # Configuration loading (Viper)
│   ├── mcp/             # MCP server wrapper (official SDK)
│   ├── providers/       # Tool providers (Databricks, IO, Workspace, Deployment)
│   ├── sandbox/         # Execution abstraction (Local, Dagger stub)
│   ├── session/         # Session state management
│   ├── logging/         # Structured logging with rotation
│   └── templates/       # Template abstraction
└── internal/templates/  # Embedded project templates
```
```

#### Section 2: Add Provider Documentation

Add after the **Workspace Provider** section (around line 114):

```markdown
**Deployment Provider** (pkg/providers/deployment)
- Tools: deploy_databricks_app
- Requires validated project (checksum verified)
- Only enabled when config.AllowDeployment = true
- Workflow: build → get/create app → sync → deploy
- Ownership checks prevent overwriting others' apps
- Retry logic (3 attempts) for deployment
- State tracking via `.edda_state` file (Validated → Deployed)
```

#### Section 3: Add Configuration Documentation

Update the Configuration section (around line 172) to include deployment settings:

```go
type Config struct {
    AllowDeployment    bool
    WithWorkspaceTools bool
    RequiredProviders  []string
    IOConfig           *IOConfig
    WarehouseID        string  // Required for deployment
}
```

#### Section 4: Add to Environment Variables

Update environment variables section (around line 367):

```bash
# Databricks credentials
DATABRICKS_HOST=https://your-workspace.databricks.com
DATABRICKS_TOKEN=dapi...
DATABRICKS_WAREHOUSE_ID=your-warehouse-id  # Required for deployment
```

#### Section 5: Add to Key Files Reference

Update the table (around line 382):

```markdown
| File | Purpose |
|------|---------|
| `pkg/providers/databricks/deployment.go` | Databricks CLI wrapper for app deployment |
| `pkg/providers/deployment/provider.go` | Deployment tool registration and workflow |
```

#### Section 6: Add Security Considerations

Add to security section (around line 262):

```markdown
**Deployment Security**:
- Deployment must be explicitly enabled via `AllowDeployment` config
- Ownership checks prevent overwriting other users' apps without `force` flag
- Checksum verification ensures no modifications since validation
- State machine enforces: Scaffolded → Validated → Deployed
- Requires `DATABRICKS_WAREHOUSE_ID` environment variable
```

### File 2: Update README.md

File: `go-mcp/README.md`

#### Section 1: Add to Features

Update features list:

```markdown
## Features

- **Databricks Integration**: Query catalogs, schemas, tables, and execute SQL
- **Project Scaffolding**: Generate full-stack TypeScript applications
- **Project Validation**: Build, type-check, and test applications
- **Deployment**: Deploy validated applications to Databricks Apps
- **Workspace Tools**: File operations, bash, grep, and glob
- **Sandboxed Execution**: Isolated file/command execution
```

#### Section 2: Add Deployment Example

Add new section after validation examples:

```markdown
### Deploying Applications

Deploy a validated application to Databricks Apps:

```json
{
  "tool": "deploy_databricks_app",
  "arguments": {
    "work_dir": "/absolute/path/to/project",
    "name": "my-data-app",
    "description": "Customer analytics dashboard",
    "force": false
  }
}
```

**Prerequisites**:
1. Application must be validated first
2. Set `allow_deployment: true` in config
3. Set `DATABRICKS_WAREHOUSE_ID` environment variable
4. Databricks CLI must be installed and configured

**Deployment Process**:
1. Verifies project is validated (checksum check)
2. Installs dependencies (`npm install`)
3. Builds frontend (`npm run build`)
4. Gets or creates Databricks app
5. Syncs workspace files
6. Deploys with retry logic (3 attempts)
7. Updates project state to Deployed
```

#### Section 3: Add Configuration Example

Update configuration section:

```json
{
  "allow_deployment": true,
  "with_workspace_tools": true,
  "required_providers": ["databricks", "io"],
  "warehouse_id": "abc123def456",
  "io_config": {
    "template": {
      "name": "default"
    }
  }
}
```

#### Section 4: Add Prerequisites

Update installation/prerequisites section:

```markdown
## Prerequisites

- Go 1.21 or later
- Databricks workspace and credentials (for Databricks integration)
- Databricks CLI installed (for deployment: `pip install databricks-cli`)
- Node.js and npm (for application scaffolding and validation)
```

### File 3: Create Example Config

Create: `go-mcp/examples/config-with-deployment.json`

```json
{
  "allow_deployment": true,
  "with_workspace_tools": true,
  "required_providers": [
    "databricks",
    "io"
  ],
  "warehouse_id": "your-warehouse-id-here",
  "io_config": {
    "template": {
      "name": "default",
      "path": ""
    },
    "validation": {
      "command": "",
      "docker_image": ""
    }
  }
}
```

### File 4: Create Deployment Guide

Create: `go-mcp/docs/DEPLOYMENT.md`

```markdown
# Deployment Guide

This guide covers deploying applications to Databricks Apps using go-mcp.

## Prerequisites

1. **Databricks CLI**: Install and configure
   ```bash
   pip install databricks-cli
   databricks configure --token
   ```

2. **Environment Variables**:
   ```bash
   export DATABRICKS_HOST=https://your-workspace.databricks.com
   export DATABRICKS_TOKEN=dapi...
   export DATABRICKS_WAREHOUSE_ID=your-warehouse-id
   ```

3. **Configuration**: Enable deployment in `~/.go-mcp/config.json`
   ```json
   {
     "allow_deployment": true,
     "warehouse_id": "your-warehouse-id"
   }
   ```

## Workflow

### 1. Scaffold Application
```json
{
  "tool": "scaffold_data_app",
  "arguments": {
    "work_dir": "/path/to/project"
  }
}
```

### 2. Develop and Customize
Edit files in `/path/to/project`:
- `client/` - Frontend React/tRPC code
- `server/` - Backend tRPC routes

### 3. Validate Application
```json
{
  "tool": "validate_data_app",
  "arguments": {
    "work_dir": "/path/to/project"
  }
}
```

This runs:
- `npm run build` - Build check
- `tsc --noEmit` - Type check
- `npm test` - Run tests

### 4. Deploy to Databricks
```json
{
  "tool": "deploy_databricks_app",
  "arguments": {
    "work_dir": "/path/to/project",
    "name": "my-app",
    "description": "My data application",
    "force": false
  }
}
```

## Deployment Process

The deployment tool:

1. **Validates State**: Checks `.edda_state` file
2. **Verifies Checksum**: Ensures no changes since validation
3. **Builds Application**: Runs `npm install` and `npm run build`
4. **Creates/Gets App**: Uses Databricks Apps API
5. **Syncs Workspace**: Uploads server files to Databricks
6. **Deploys**: Deploys with retry logic (3 attempts)
7. **Updates State**: Marks project as Deployed

## Security Features

### Ownership Checks
- By default, prevents overwriting apps created by other users
- Use `force: true` to override (use with caution)

### Checksum Verification
- Deployment fails if files changed since validation
- Re-run validation after changes

### State Machine
Project must progress through states:
- **Scaffolded** → **Validated** → **Deployed**

Cannot deploy without validation first.

## Troubleshooting

### Error: "Project must be validated before deployment"
**Solution**: Run `validate_data_app` first

### Error: "DATABRICKS_WAREHOUSE_ID environment variable is required"
**Solution**: Set the warehouse ID:
```bash
export DATABRICKS_WAREHOUSE_ID=your-warehouse-id
```

### Error: "App already exists and was created by another user"
**Cause**: Another user created an app with the same name
**Solutions**:
1. Choose a different name
2. Use `force: true` (if you have permission)
3. Contact the app owner

### Error: "Project files changed since validation"
**Solution**: Re-run `validate_data_app`

### Deployment Fails After Sync
**Cause**: Databricks Apps deployment can be flaky
**Solution**: The tool automatically retries 3 times. If still failing:
1. Check Databricks workspace logs
2. Verify workspace path is accessible
3. Check app configuration in Databricks UI

## Best Practices

1. **Always Validate First**: Never skip validation before deployment
2. **Use Meaningful Names**: App names should be descriptive and unique
3. **Test Locally**: Use workspace tools to test before deploying
4. **Monitor State**: Check `.edda_state` file for project status
5. **Keep Warehouse ID Secure**: Don't commit credentials to version control

## State File Format

The `.edda_state` file tracks project lifecycle:

```json
{
  "state": "Deployed",
  "data": {
    "validated_at": "2024-01-15T10:30:00Z",
    "checksum": "abc123...",
    "deployed_at": "2024-01-15T10:45:00Z"
  }
}
```

## CLI Commands

Check app status using Databricks CLI:

```bash
# List apps
databricks apps list

# Get app details
databricks apps get my-app

# View app in browser
databricks apps open my-app
```
```

## Verification Checklist

After updating documentation:
- [ ] CLAUDE.md updated with deployment provider info
- [ ] README.md includes deployment examples
- [ ] Example config file created
- [ ] Deployment guide created
- [ ] All code examples tested
- [ ] Links between docs work correctly
- [ ] Formatting is consistent
- [ ] No typos or grammatical errors

## Success Criteria
- [ ] All documentation files updated
- [ ] Examples are clear and accurate
- [ ] Deployment guide is comprehensive
- [ ] Configuration examples are correct
- [ ] Troubleshooting section addresses common issues
- [ ] Security considerations documented
- [ ] Prerequisites clearly stated
