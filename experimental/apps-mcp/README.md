# Databricks MCP Server

A Model Context Protocol (MCP) server for working with Databricks through natural language. This server provides tools for data exploration, workspace management, and executing Databricks CLI commands through AI-powered conversations.

## TL;DR

**Primary Goal:** Interact with Databricks workspaces, manage Databricks Asset Bundles (DABs), deploy Databricks Apps, and query data through natural language conversations.

**How it works:**
1. **Explore your workspace** - Discover workspace resources, get CLI command examples, and workflow recommendations
2. **Query your data** - Browse catalogs, schemas, and tables; execute SQL queries via CLI commands
3. **Manage bundles** - Initialize, validate, deploy, and run Databricks Asset Bundles
4. **Deploy apps** - Deploy and manage Databricks Apps through CLI commands
5. **Execute any CLI command** - Run the full Databricks CLI through the `invoke_databricks_cli` tool

**Why use it:**
- **Conversational interface**: Work with Databricks using natural language instead of memorizing CLI commands
- **Context-aware**: Get relevant command suggestions based on your workspace configuration
- **Unified workflow**: Combine data exploration, bundle management, and app deployment in one tool

Perfect for data engineers and developers who want to streamline their Databricks workflows with AI-powered assistance.

---

## Getting Started

### Quick Setup

1. **Set up Databricks credentials** (required for Databricks tools):
   ```bash
   export DATABRICKS_HOST="https://your-workspace.databricks.com"
   export DATABRICKS_TOKEN="dapi..."
   export DATABRICKS_WAREHOUSE_ID="your-warehouse-id"
   ```

2. **Configure your MCP client** (e.g., Claude Code):

   Add to your MCP config file (e.g., `~/.claude.json`):
   ```json
   {
     "mcpServers": {
       "databricks": {
         "command": "databricks",
         "args": ["experimental", "apps-mcp"],
         "env": {
           "DATABRICKS_HOST": "https://your-workspace.databricks.com",
           "DATABRICKS_TOKEN": "dapi...",
           "DATABRICKS_WAREHOUSE_ID": "your-warehouse-id"
         }
       }
     }
   }
   ```

3. **Start using Databricks with natural language:**

   Restart your MCP client and try:
   ```
   Explore my Databricks workspace and show me what catalogs are available
   ```

   ```
   Initialize a new Databricks Asset Bundle for a data pipeline project
   ```

   ```
   Query the main.sales.transactions table and show me the top 10 customers by revenue
   ```

   The AI will use the appropriate Databricks tools to help you complete these tasks.

---

## Features

The Databricks MCP server provides CLI-based tools for workspace interaction:

Execute Databricks CLI commands and explore workspace resources:

- **`explore`** - Discover workspace resources and get CLI command recommendations
  - Lists workspace URL, SQL warehouse details, and authentication profiles
  - Provides command examples for jobs, clusters, catalogs, tables, and workspace files
  - Gives workflow guidance for Databricks Asset Bundles and Apps

- **`invoke_databricks_cli`** - Execute any Databricks CLI command
  - Run MCP app commands: `experimental apps-mcp tools init-template`, `bundle validate`, `bundle deploy`, `bundle run`
  - Run apps commands: `apps deploy`, `apps list`, `apps get`, `apps start`, `apps stop`
  - Run workspace commands: `workspace list`, `workspace export`, `jobs list`, `clusters list`
  - Run catalog commands: `catalogs list`, `schemas list`, `tables list`
  - Supports all Databricks CLI functionality with proper user allowlisting

*These tools provide a conversational interface to the full Databricks CLI, including Unity Catalog exploration and SQL query execution.*

---

## Example Usage

Here are example conversations showing common workflows:

### Data Exploration

**Explore workspace resources:**
```
Explore my Databricks workspace and show me what's available
```

**Query data:**
```
Show me the schema of the main.sales.transactions table and give me a sample of 10 rows
```

**Find specific tables:**
```
Find all tables in the main catalog that contain the word "customer"
```

### Databricks Asset Bundles (DABs)

**Create a new bundle project:**
```
Initialize a new Databricks Asset Bundle for a data pipeline project
```

**Deploy a bundle:**
```
Validate and deploy my Databricks bundle to the dev environment
```

**Run a job from a bundle:**
```
Run the data_processing job from my bundle
```

### Databricks Apps

**Initialize an app from template:**
```
Initialize a new Streamlit app using the Databricks bundle template
```

**Deploy an app:**
```
Deploy my app in the current directory to Databricks Apps as "sales-dashboard"
```

**Manage apps:**
```
List all my Databricks Apps and show me their status
```

### Working with Jobs and Clusters

**List and inspect jobs:**
```
Show me all jobs in the workspace and their recent run status
```

**Get cluster details:**
```
List all clusters and show me the configuration of the production cluster
```

### Complex Workflows

**End-to-end data pipeline:**
```
1. Show me what tables are in the main.raw catalog
2. Create a new bundle for an ETL pipeline
3. Deploy it to the dev environment
4. Run the pipeline and show me the results
```

**Multi-environment deployment:**
```
Validate my bundle, then deploy it to dev, staging, and production environments
```

---

## Benefits

### Natural Language Interface

Instead of memorizing complex CLI commands and flags, you can:
- Ask questions in plain English
- Get context-aware command suggestions
- Execute commands through conversation
- Receive explanations of results

### Workspace Awareness

The `explore` tool provides:
- Automatic workspace configuration detection
- SQL warehouse information
- Authentication profile details
- Relevant command examples based on your setup

### Unified Workflow

Work with all Databricks functionality from one place:
- **Data exploration**: Query catalogs, schemas, and tables
- **Bundle management**: Create, validate, and deploy DABs
- **App deployment**: Deploy and manage Databricks Apps
- **Workspace operations**: Manage jobs, clusters, and notebooks

### Safe Command Execution

The `invoke_databricks_cli` tool:
- Allows users to allowlist specific commands
- Provides better tracking of executed operations
- Maintains audit trail of AI actions
- Prevents unauthorized operations

---

## Reference

### CLI Commands

```bash
# Start MCP server (default mode)
databricks experimental apps-mcp --warehouse-id <warehouse-id>

# Enable workspace tools
databricks experimental apps-mcp --warehouse-id <warehouse-id> --with-workspace-tools
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--warehouse-id` | Databricks SQL Warehouse ID (required for SQL queries) | - |
| `--with-workspace-tools` | Enable workspace file operations | `false` |
| `--help` | Show help | - |

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABRICKS_HOST` | Databricks workspace URL | `https://your-workspace.databricks.com` |
| `DATABRICKS_TOKEN` | Databricks personal access token | `dapi...` |
| `WAREHOUSE_ID` | Databricks SQL warehouse ID (preferred) | `abc123def456` |
| `DATABRICKS_WAREHOUSE_ID` | Alternative name for warehouse ID | `abc123def456` |
| `WITH_WORKSPACE_TOOLS` | Enable workspace tools | `true` or `false` |

### Authentication

The MCP server uses standard Databricks CLI authentication methods:

1. **Environment variables** (as shown in the config above)
2. **Databricks CLI profiles** - Use `--profile` flag or `DATABRICKS_PROFILE` env var
3. **Default profile** - Uses `~/.databrickscfg` default profile if available

For more details, see the [Databricks authentication documentation](https://docs.databricks.com/en/dev-tools/cli/authentication.html).

### Requirements

- **Databricks CLI** (this package)
- **Databricks workspace** with a SQL warehouse
- **MCP-compatible client** (Claude Desktop, Continue, etc.)

---

## License

See the main repository license.

## Contributing

Contributions welcome! Please see the main repository for development guidelines.

## Support

- **Issues**: https://github.com/databricks/cli/issues
- **Documentation**: https://docs.databricks.com/dev-tools/cli/databricks-cli.html
