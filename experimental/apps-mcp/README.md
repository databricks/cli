# Databricks MCP Server

A Model Context Protocol (MCP) server for generating production-ready Databricks applications with testing,
linting and deployment setup from a single prompt. This agent relies heavily on scaffolding and
extensive validation to ensure high-quality outputs.

## TL;DR

**Primary Goal:** Create and deploy production-ready Databricks applications from a single natural language prompt. This MCP server combines scaffolding, validation, and deployment into a seamless workflow that goes from idea to running application.

**How it works:**
1. **Explore your data** - Query Databricks catalogs, schemas, and tables to understand your data
2. **Generate the app** - Scaffold a full-stack TypeScript application (tRPC + React) with proper structure
3. **Customize with AI** - Use workspace tools to read, write, and edit files naturally through conversation
4. **Validate rigorously** - Run builds, type checks, and tests to ensure quality
5. **Deploy confidently** - Push validated apps directly to Databricks Apps platform

**Why use it:**
- **Speed**: Go from concept to deployed Databricks app in minutes, not hours or days
- **Quality**: Extensive validation ensures your app builds, passes tests, and is production-ready
- **Simplicity**: One natural language conversation handles the entire workflow

Perfect for data engineers and developers who want to build Databricks apps without the manual overhead of project setup, configuration, testing infrastructure, and deployment pipelines.

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

3. **Create your first Databricks app:**

   Restart your MCP client and try:
   ```
   Create a Databricks app that shows sales data from main.sales.transactions
   with a chart showing revenue by region. Deploy it as "sales-dashboard".
   ```

   The AI will:
   - Explore your Databricks tables
   - Generate a full-stack application
   - Customize it based on your requirements
   - Validate it passes all tests
   - Deploy it to Databricks Apps

---

## Features

All features are designed to support the end-to-end workflow of creating production-ready Databricks applications:

### 1. Data Exploration (Foundation)

Understand your Databricks data before building:

- **`databricks_list_catalogs`** - Discover available data catalogs
- **`databricks_list_schemas`** - Browse schemas in a catalog
- **`databricks_list_tables`** - Find tables in a schema
- **`databricks_describe_table`** - Get table details, columns, and sample data
- **`databricks_execute_query`** - Test queries and preview data

*These tools help the AI understand your data structure so it can generate relevant application code.*

### 2. Application Generation (Core)

Create the application structure:

- **`scaffold_data_app`** - Generate a full-stack TypeScript application
  - Modern stack: Node.js, TypeScript, React, tRPC
  - Pre-configured build system, linting, and testing
  - Production-ready project structure
  - Databricks SDK integration

*This is the foundation of your application - a working, tested template ready for customization.*

### 3. Validation (Quality Assurance)

Ensure production-readiness before deployment:

- **`validate_data_app`** - Comprehensive validation
  - Build verification (npm build)
  - Type checking (TypeScript compiler)
  - Test execution (full test suite)

*This step guarantees your application is tested and ready for production before deployment.*

### 4. Deployment (Production Release)

Deploy validated applications to Databricks (enable with `--allow-deployment`):

- **`deploy_databricks_app`** - Push to Databricks Apps platform
  - Automatic deployment configuration
  - Environment management
  - Production-grade setup

*The final step: your validated application running on Databricks.*

---

## Example Usage

Here are example conversations showing the end-to-end workflow for creating Databricks applications:

### Complete Workflow: Analytics Dashboard

This example shows how to go from data exploration to deployed application:

**User:**
```
I want to create a Databricks app that visualizes customer purchases. The data is
in the main.sales catalog. Show me what tables are available and create a dashboard
with charts for total revenue by region and top products. Deploy it as "sales-insights".
```

**What happens:**
1. **Data Discovery** - AI lists schemas and tables in main.sales
2. **Data Inspection** - AI describes the purchases table structure
3. **App Generation** - AI scaffolds a TypeScript application
4. **Customization** - AI adds visualization components and queries
5. **Validation** - AI runs build, type check, and tests in container
6. **Deployment** - AI deploys to Databricks Apps as "sales-insights"

**Result:** A production-ready Databricks app running in minutes with proper testing.

---

### Quick Examples for Specific Use Cases

#### Data App from Scratch

```
Create a Databricks app in ~/projects/user-analytics that shows daily active users
from main.analytics.events. Include a line chart and data table.
```

#### Real-Time Monitoring Dashboard

```
Build a monitoring dashboard for the main.logs.system_metrics table. Show CPU,
memory, and disk usage over time. Add alerts for values above thresholds.
```

#### Report Generator

```
Create an app that generates weekly reports from main.sales.transactions.
Include revenue trends, top customers, and product performance. Add export to CSV.
```

#### Data Quality Dashboard

```
Build a data quality dashboard for main.warehouse.inventory. Check for nulls,
duplicates, and out-of-range values. Show data freshness metrics.
```

---

### Working with Existing Applications

Once an app is scaffolded, you can continue development through conversation:

```
Add a filter to show only transactions from the last 30 days
```

```
Update the chart to use a bar chart instead of line chart
```

```
Add a new API endpoint to fetch customer details
```

```
Run the tests and fix any failures
```

```
Add error handling for failed database queries
```

---

### Iterative Development Workflow

**Initial Request:**
```
Create a simple dashboard for main.sales.orders
```

**Refinement:**
```
Add a date range picker to filter orders
```

**Enhancement:**
```
Include a summary card showing total orders and revenue
```

**Quality Check:**
```
Validate the app and show me any test failures
```

**Production:**
```
Deploy the app to Databricks as "orders-dashboard"
```

---

## Why This Approach Works

### Traditional Development vs. Databricks MCP

| Traditional Approach | With Databricks MCP |
|---------------------|-------------|
| Manual project setup (hours) | Instant scaffolding (seconds) |
| Configure build tools manually | Pre-configured and tested |
| Set up testing infrastructure | Built-in test suite |
| Manual code changes and debugging | AI-powered development with validation |
| Local testing only | Containerized validation (reproducible) |
| Manual deployment setup | Automated deployment to Databricks |
| **Time to production: days/weeks** | **Time to production: minutes** |

### Key Advantages

**1. Scaffolding + Validation = Quality**
- Start with a working, tested template
- Every change is validated before deployment
- No broken builds reach production

**2. Natural Language = Productivity**
- Describe what you want, not how to build it
- AI handles implementation details
- Focus on requirements, not configuration

**3. End-to-End Workflow = Simplicity**
- Single tool for entire lifecycle
- No context switching between tools
- Seamless progression from idea to deployment

### What Makes It Production-Ready

The Databricks MCP server doesn't just generate code—it ensures quality:

- ✅ **TypeScript** - Type safety catches errors early
- ✅ **Build verification** - Ensures code compiles
- ✅ **Test suite** - Validates functionality
- ✅ **Linting** - Enforces code quality
- ✅ **Databricks integration** - Native SDK usage

---

## Reference

### CLI Commands

```bash
# Start MCP server (default mode)
databricks experimental apps-mcp --warehouse-id <warehouse-id>

# Enable workspace tools
databricks experimental apps-mcp --warehouse-id <warehouse-id> --with-workspace-tools

# Enable deployment
databricks experimental apps-mcp --warehouse-id <warehouse-id> --allow-deployment
```

### CLI Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--warehouse-id` | Databricks SQL Warehouse ID (required) | - |
| `--with-workspace-tools` | Enable workspace file operations | `false` |
| `--allow-deployment` | Enable deployment operations | `false` |
| `--help` | Show help | - |

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `DATABRICKS_HOST` | Databricks workspace URL | `https://your-workspace.databricks.com` |
| `DATABRICKS_TOKEN` | Databricks personal access token | `dapi...` |
| `WAREHOUSE_ID` | Databricks SQL warehouse ID (preferred) | `abc123def456` |
| `DATABRICKS_WAREHOUSE_ID` | Alternative name for warehouse ID | `abc123def456` |
| `ALLOW_DEPLOYMENT` | Enable deployment operations | `true` or `false` |
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
