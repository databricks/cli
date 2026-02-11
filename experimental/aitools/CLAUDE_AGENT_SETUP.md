# Using the Agent Template with Claude Code

This guide explains how to configure Claude Code to use the Databricks CLI from source and scaffold agent projects with the experimental agent template.

## Quick Start

### Prerequisites

1. **Databricks CLI built from source** at `~/cli/cli`
2. **Claude Code** (or compatible MCP-enabled IDE)
3. **Databricks workspace** with authentication configured

### Installation Steps

#### 1. Configure Claude Code MCP Server

Edit your Claude Code configuration file:

**Location:** `~/Library/Application Support/Claude/claude_desktop_config.json`

**Add the Databricks MCP server:**

```json
{
  "mcpServers": {
    "databricks-mcp": {
      "command": "/Users/YOUR_USERNAME/cli/cli",
      "args": [
        "experimental",
        "aitools",
        "--warehouse-id",
        "YOUR_WAREHOUSE_ID"
      ]
    }
  }
}
```

**Important:**
- Replace `YOUR_USERNAME` with your actual username
- Replace `YOUR_WAREHOUSE_ID` with your SQL warehouse ID (find in workspace or use default from profile)

**Why this works:** The MCP server uses `os.Args[0]` to determine which CLI binary to invoke for all subsequent commands. By starting the MCP server with `~/cli/cli`, all CLI operations automatically use the source version.

#### 2. Restart Claude Code

After updating the configuration, **completely restart Claude Code** for changes to take effect.

#### 3. Verify Installation

In a new Claude Code session:

1. Ask Claude to call `databricks_discover` with your working directory
2. Verify the output shows available commands including:
   ```
   experimental aitools tools init-template agent --name my-agent
   ```

#### 4. Authenticate with Workspace

Before creating projects, authenticate:

```bash
~/cli/cli auth login --profile dogfood
```

Or ask Claude to use the `databricks_configure_auth` MCP tool with your profile and host.

## Creating Agent Projects

### Using Local Template (Development)

When testing local template changes:

```bash
export DATABRICKS_AGENT_TEMPLATE_PATH="/Users/YOUR_USERNAME/cli/experimental/aitools/templates/agent-openai-agents-sdk"
~/cli/cli --profile dogfood experimental aitools tools init-template agent \
  --name my_agent \
  --output-dir ~/my-agent-app
```

### Using Remote Template (Production)

Once the template is merged to main:

```bash
~/cli/cli --profile dogfood experimental aitools tools init-template agent \
  --name my_agent \
  --output-dir ~/my-agent-app
```

The template will be automatically fetched from GitHub.

### Via Claude Code

Ask Claude:

```
Create a new Databricks agent app called "my_agent" using the agent template
```

Claude will:
1. Use the `invoke_databricks_cli` MCP tool
2. Set `DATABRICKS_AGENT_TEMPLATE_PATH` if testing local changes
3. Run the init-template command
4. Scaffold the project with all necessary files

## What You Get

The agent template creates a complete project with:

```
my-agent-app/
├── agent_server/
│   ├── agent.py              # Agent logic, model config, MCP servers
│   ├── start_server.py       # FastAPI server with MLflow tracing
│   ├── evaluate_agent.py     # Agent evaluation framework
│   └── utils.py              # Helper functions
├── scripts/
│   ├── quickstart.sh         # First-time setup script
│   └── start_app.py          # Development server manager
├── databricks.yml            # Bundle and app configuration
├── app.yaml                  # App environment variables
├── pyproject.toml            # Python dependencies (uv)
├── AGENTS.md                 # Comprehensive development guide
├── CLAUDE.md                 # Claude Code integration
└── README.md                 # Quick start guide
```

### Key Features

- **OpenAI Agents SDK** - Build conversational agents
- **Claude 3.7 Sonnet** - Pre-configured Databricks-hosted model
- **MCP integration** - Access to Unity Catalog functions (code interpreter)
- **MLflow tracing** - Automatic trace capture with `@invoke()` and `@stream()` decorators
- **Streaming support** - Both streaming and non-streaming endpoints
- **Evaluation framework** - Built-in MLflow scorers for agent quality

## Next Steps After Scaffolding

### Local Development

```bash
cd my-agent-app
./scripts/quickstart.sh   # Setup: auth, MLflow experiment, dependencies
uv run start-app          # Start server at http://localhost:8000
```

### Test Locally

```bash
# Streaming
curl -X POST http://localhost:8000/invocations \
  -H "Content-Type: application/json" \
  -d '{"input": [{"role": "user", "content": "hi"}], "stream": true}'

# Non-streaming
curl -X POST http://localhost:8000/invocations \
  -H "Content-Type: application/json" \
  -d '{"input": [{"role": "user", "content": "hi"}]}'
```

### Deploy to Databricks

```bash
# Create MLflow experiment
~/cli/cli --profile dogfood experiments create-experiment \
  /Users/$USER/my-agent-app

# Deploy bundle with experiment ID
~/cli/cli --profile dogfood bundle deploy \
  --var mlflow_experiment_id=<experiment_id>

# Get app URL
~/cli/cli --profile dogfood bundle summary
```

## Troubleshooting

### MCP Server Not Found

**Symptom:** Claude can't find Databricks tools

**Solution:**
1. Verify `claude_desktop_config.json` has correct path to `~/cli/cli`
2. Check that `~/cli/cli` exists and is executable
3. Restart Claude Code completely

### Template Not Available

**Symptom:** Command says "unknown template" or can't find agent template

**Solution:**
1. Ensure CLI was built with template: `cd ~/cli && make build`
2. Verify template exists: `ls ~/cli/experimental/aitools/templates/agent-openai-agents-sdk`
3. Check `databricks_template_schema.json` exists in template directory
4. Use `DATABRICKS_AGENT_TEMPLATE_PATH` environment variable for local template

### Authentication Issues

**Symptom:** "Invalid token" or "Not authenticated"

**Solution:**
1. Run: `~/cli/cli auth login --profile <profile-name>`
2. Or use Claude's `databricks_configure_auth` tool with profile and host
3. Verify: `~/cli/cli current-user me --profile <profile-name>`

### App Name Validation Errors

**Symptom:** "App name must contain only lowercase letters, numbers, and dashes"

**Solution:**
- Use only lowercase letters, numbers, and dashes in app names
- ✅ Good: `my-agent-app`
- ❌ Bad: `my_agent_app` (underscores not allowed)
- ❌ Bad: `MyAgentApp` (uppercase not allowed)

## Understanding the MCP Server Architecture

### How CLI Path Resolution Works

```go
// In experimental/aitools/lib/common/common.go
func GetCLIPath() string {
    return os.Args[0]  // Returns the binary that started the process
}
```

When you configure Claude Code with:
```json
{
  "command": "/Users/YOUR_USERNAME/cli/cli",
  "args": ["experimental", "aitools", "--warehouse-id", "..."]
}
```

The MCP server starts with `os.Args[0] = /Users/YOUR_USERNAME/cli/cli`

Then every CLI invocation uses:
```go
cmd := exec.CommandContext(ctx, cliPath, args...)  // cliPath = os.Args[0]
```

This ensures all CLI commands run from source automatically!

### Why This Matters for Development

**Without source CLI:**
- Changes to templates require CLI release
- Can't test experimental features
- Limited to stable, released functionality

**With source CLI:**
- Immediate access to template changes
- Test experimental features before release
- Iterate quickly on template improvements
- Override templates with local versions via environment variables

## Environment Variable Overrides

### Agent Template

```bash
export DATABRICKS_AGENT_TEMPLATE_PATH="/path/to/local/agent-template"
```

This tells the agent command to use your local template instead of fetching from GitHub.

### App Template

```bash
export DATABRICKS_APPKIT_TEMPLATE_PATH="/path/to/local/appkit-template"
```

This tells the app command to use your local template instead of fetching from GitHub.

### When to Use

**Local development:**
- Testing template changes before committing
- Iterating on template structure
- Debugging template issues
- Adding new features to templates

**Not needed when:**
- Using released/merged templates
- Template is already in your CLI build
- Working with stable templates (job, pipeline, empty)

## Advanced: Multiple CLI Versions

If you need to work with multiple CLI versions:

### Option 1: Multiple MCP Servers

```json
{
  "mcpServers": {
    "databricks-mcp-source": {
      "command": "/Users/YOUR_USERNAME/cli/cli",
      "args": ["experimental", "aitools", "--warehouse-id", "..."]
    },
    "databricks-mcp-stable": {
      "command": "/usr/local/bin/databricks",
      "args": ["experimental", "aitools", "--warehouse-id", "..."]
    }
  }
}
```

Claude will have access to both and can use either based on context.

### Option 2: Swap Configuration

Keep different config files and swap them:

```bash
# Use source CLI
cp ~/.config/claude-source.json ~/Library/Application\ Support/Claude/claude_desktop_config.json

# Use stable CLI
cp ~/.config/claude-stable.json ~/Library/Application\ Support/Claude/claude_desktop_config.json
```

Restart Claude Code after swapping.

## Template Development Workflow

### 1. Make Template Changes

Edit files in:
```
~/cli/experimental/aitools/templates/agent-openai-agents-sdk/
```

### 2. Build CLI (if code changes)

```bash
cd ~/cli
make build
```

**Note:** Template file changes don't require rebuild - they're read at runtime.

### 3. Test with Local Override

```bash
export DATABRICKS_AGENT_TEMPLATE_PATH="$HOME/cli/experimental/aitools/templates/agent-openai-agents-sdk"
~/cli/cli experimental aitools tools init-template agent --name test-agent --output-dir /tmp/test
```

### 4. Verify Scaffolded Project

```bash
cd /tmp/test/test-agent
ls -la
cat databricks.yml
```

### 5. Test Local Development

```bash
./scripts/quickstart.sh
uv run start-app
# Test at http://localhost:8000
```

### 6. Commit and Push

```bash
cd ~/cli
git add experimental/aitools/templates/agent-openai-agents-sdk/
git commit -m "Update agent template: [description]"
git push
```

### 7. Test Without Override

After merging to main, test without environment variable:

```bash
unset DATABRICKS_AGENT_TEMPLATE_PATH
~/cli/cli experimental aitools tools init-template agent --name test-agent2 --output-dir /tmp/test2
```

This verifies the template works when fetched from GitHub.

## Comparison: Experimental vs Stable Templates

| Feature | Experimental (agent, app) | Stable (job, pipeline, empty) |
|---------|--------------------------|------------------------------|
| Location | `experimental/aitools/templates/` | `libs/template/templates/` |
| Source | GitHub (fetched) | CLI binary (bundled) |
| Override | Environment variable | Not needed |
| Evolution | Rapid, experimental | Stable, production-ready |
| CLI flag | `experimental aitools tools` | `bundle init` |
| Use case | Development, testing | Production, stable features |

## References

- **MCP Server Implementation:** `experimental/aitools/lib/providers/clitools/`
- **CLI Path Resolution:** `experimental/aitools/lib/common/common.go`
- **Agent Template Command:** `experimental/aitools/cmd/init_template/agent.go`
- **Agent Template Files:** `experimental/aitools/templates/agent-openai-agents-sdk/`
- **Template Schema:** `databricks_template_schema.json` in each template
- **Architecture Decisions:** `experimental/aitools/AGENT_TEMPLATE_ARCHITECTURE.md`
