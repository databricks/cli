# Agent OpenAI Agents SDK Development Guide

## For AI Agents: MANDATORY First Action

**BEFORE any other action, call `mcp__databricks-mcp__databricks_discover` with this project's path.**

This is required because the discover tool:
- Detects project state (new, deployed, running) and provides relevant guidance
- Shows available CLI commands and recommended next steps
- Configures workspace authentication

Without this context, you cannot effectively help with this project.

If the tool is unavailable, the user needs to set up the Databricks CLI MCP server:
1. Install CLI: https://docs.databricks.com/dev-tools/cli/install
2. Run: `databricks experimental aitools install`

---

## Running the App

**Prerequisites:** uv, nvm (Node 20), Databricks CLI

**Quick Start:**

```bash
./scripts/quickstart.sh   # First-time setup (auth, MLflow experiment, env)
uv run start-app          # Start app at http://localhost:8000
```

**Advanced Server Options:**

```bash
uv run start-server --reload   # Hot-reload on code changes during development
uv run start-server --port 8001
uv run start-server --workers 4
```

**Test API:**

```bash
# Streaming request
curl -X POST http://localhost:8000/invocations \
  -H "Content-Type: application/json" \
  -d '{ "input": [{ "role": "user", "content": "hi" }], "stream": true }'

# Non-streaming request
curl -X POST http://localhost:8000/invocations \
  -H "Content-Type: application/json" \
  -d '{ "input": [{ "role": "user", "content": "hi" }] }'
```

---

## Testing the Agent

**Run evaluation:**

```bash
uv run agent-evaluate     # Uses MLflow scorers (RelevanceToQuery, Safety)
```

**Run unit tests:**

```bash
pytest [path]             # Standard pytest execution
```

---

## Modifying the Agent

Anytime the user wants to modify the agent, look through each of the following resources to help them accomplish their goal:

If the user wants to convert something into Responses API, refer to https://mlflow.org/docs/latest/genai/serving/responses-agent/ for more information.

1. Look through existing databricks-openai APIs to see if they can use one of these to accomplish their goal.
2. Look through the folders in https://github.com/bbqiu/agent-on-app-prototype to see if there's an existing example similar to what they're looking to do.
3. Reference the documentation available under https://docs.databricks.com/aws/en/generative-ai/agent-framework/ and its subpages.
4. For adding tools and capabilities, refer to: https://docs.databricks.com/aws/en/generative-ai/agent-framework/agent-tool
5. Reference the OpenAI Agents SDK documentation: https://platform.openai.com/docs/guides/agents-sdk

**Main file to modify:** `agent_server/agent.py`

---

## databricks-openai SDK Overview

**SDK Location:** `https://github.com/databricks/databricks-ai-bridge/tree/main/integrations/openai`

**Development Workflow:**

```bash
uv add databricks-openai
```

Before making any changes, ensure that the APIs actually exist in the SDK. If something is missing from the documentation here, feel free to look in the venv's `site-packages` directory for the `databricks_openai` package. If it's not installed, run `uv sync` in this folder to create the .venv and install the package.

---

### Key Components

#### 1. MCP Servers - Tool Integration

Connect to MCP (Model Context Protocol) servers to get tools for your agent.

**Basic MCP Server:**

```python
from databricks_openai.agents import McpServer

async def init_mcp_server():
    return McpServer(
        url=f"{host}/api/2.0/mcp/functions/system/ai",
        name="system.ai uc function mcp server",
    )

# Use in agent
agent = Agent(
    name="code execution agent",
    instructions="You are a code execution agent.",
    model="databricks-claude-3-7-sonnet",
    mcp_servers=[mcp_server],
)
```

#### 2. Sync and Async Databricks OpenAI Clients

Set up Databricks-hosted OpenAI-compatible models:

```python
from databricks_openai import AsyncDatabricksOpenAI, DatabricksOpenAI
from agents import set_default_openai_api, set_default_openai_client

# Async client (recommended for agent servers)
set_default_openai_client(AsyncDatabricksOpenAI())
set_default_openai_api("chat_completions")

# Sync client
client = DatabricksOpenAI()
```

**Note:** This works for all Databricks models except GPT-OSS, which uses a slightly different API.

---

## Agent Development Patterns

### Creating and Running Agents

```python
from agents import Agent, Runner
from databricks_openai.agents import McpServer

# Create agent with MCP servers
agent = Agent(
    name="code execution agent",
    instructions="You are a code execution agent.",
    model="databricks-claude-3-7-sonnet",
    mcp_servers=[mcp_server],
)

# Run agent (non-streaming)
messages = [{"role": "user", "content": "hi"}]
result = await Runner.run(agent, messages)

# Run agent (streaming)
result = Runner.run_streamed(agent, input=messages)
async for event in result.stream_events():
    # Process stream events
    pass
```

### MLflow Tracing with Decorators

The template uses MLflow's `@invoke()` and `@stream()` decorators for automatic tracing:

```python
from mlflow.genai.agent_server import invoke, stream
from mlflow.types.responses import (
    ResponsesAgentRequest,
    ResponsesAgentResponse,
    ResponsesAgentStreamEvent,
)

@invoke()
async def invoke(request: ResponsesAgentRequest) -> ResponsesAgentResponse:
    # Agent logic here
    pass

@stream()
async def stream(request: dict) -> AsyncGenerator[ResponsesAgentStreamEvent, None]:
    # Streaming agent logic here
    pass
```

### Adding Custom Tracing

Beyond built-in tracing, you can add additional instrumentation:

```python
import mlflow

mlflow.openai.autolog()  # Auto-trace OpenAI calls

# For more granular tracing, see:
# https://docs.databricks.com/aws/en/mlflow3/genai/tracing/app-instrumentation/
```

---

## Authentication Setup

**Option 1: OAuth (Recommended)**

```bash
databricks auth login
```

Set in `.env.local`:

```bash
DATABRICKS_CONFIG_PROFILE=DEFAULT
```

**Option 2: Personal Access Token**

Set in `.env.local`:

```bash
DATABRICKS_HOST="https://host.databricks.com"
DATABRICKS_TOKEN="dapi_token"
```

---

## MLflow Experiment Setup

Create and link an MLflow experiment:

```bash
DATABRICKS_USERNAME=$(databricks current-user me | jq -r .userName)
databricks experiments create-experiment /Users/$DATABRICKS_USERNAME/agents-on-apps
```

Add the experiment ID to `.env.local`:

```bash
MLFLOW_EXPERIMENT_ID=<your-experiment-id>
```

---

## Key Files

| File                             | Purpose                                       |
| -------------------------------- | --------------------------------------------- |
| `agent_server/agent.py`          | Agent logic, model, instructions, MCP servers |
| `agent_server/start_server.py`   | FastAPI server + MLflow setup                 |
| `agent_server/evaluate_agent.py` | Agent evaluation with MLflow scorers          |
| `agent_server/utils.py`          | Databricks auth helpers, stream processing    |
| `scripts/start_app.py`           | Manages backend+frontend startup              |

---

## Deploying to Databricks Apps

**Note:** This deployment approach requires the upcoming Databricks CLI release with experiment resource support. The examples use `databricks` as the CLI command, which will work once the feature is released.

### Prerequisites

1. **Databricks CLI** with OAuth authentication configured
2. **Workspace Configuration** in `databricks.yml`

### Initial Setup

**1. Configure OAuth Authentication:**

```bash
# Login to your workspace with OAuth (required for apps)
databricks auth login --host https://your-workspace.cloud.databricks.com --profile <profile-name>

# Verify authentication
databricks current-user me --profile <profile-name>
```

**2. Configure `databricks.yml`:**

Ensure your `databricks.yml` defines both the MLflow experiment and app with resources:

```yaml
bundle:
  name: my-agent

resources:
  # MLflow experiment for agent tracing - bundle will create this
  experiments:
    agent-experiment:
      name: /Users/${workspace.current_user.userName}/${bundle.name}-${bundle.target}

  # Databricks App
  apps:
    my-agent:
      name: "${bundle.target}-my-agent"
      description: "OpenAI Agents SDK agent application"
      source_code_path: ./

      # Grant app access to the experiment
      resources:
        - name: 'experiment'
          experiment:
            experiment_id: "${resources.experiments.agent-experiment.id}"
            permission: 'CAN_MANAGE'

targets:
  dev:
    mode: development
    default: true
    workspace:
      host: https://your-workspace.cloud.databricks.com
```

**3. Configure `app.yaml`:**

Reference the experiment resource:

```yaml
env:
  - name: MLFLOW_EXPERIMENT_ID
    valueFrom: "experiment"
  - name: MLFLOW_TRACKING_URI
    value: "databricks"
  - name: MLFLOW_REGISTRY_URI
    value: "databricks-uc"
```

**4. Configure `.env.local` for Local Development:**

```bash
DATABRICKS_CONFIG_PROFILE=<profile-name>
API_PROXY=http://localhost:8000/invocations
CHAT_APP_PORT=3000
CHAT_PROXY_TIMEOUT_SECONDS=300
MLFLOW_TRACKING_URI="databricks"
MLFLOW_REGISTRY_URI="databricks-uc"
```

### Deployment Workflow

**Deploy the bundle:**

```bash
# Deploy bundle - creates experiment and grants app permissions automatically
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks bundle deploy --target dev

# Deploy the source code
DATABRICKS_USERNAME=$(databricks current-user me --profile <profile-name> | jq -r .userName)
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps deploy <app-name> \
  --source-code-path /Workspace/Users/$DATABRICKS_USERNAME/.bundle/<bundle-name>/dev/files
```

**Get app URL:**

```bash
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps get <app-name> --output json | jq -r '.url'
```

**What happens during deployment:**
- ✅ Bundle creates the MLflow experiment at `/Users/<your-username>/my-agent-dev`
- ✅ App is created with automatic access to the experiment
- ✅ App service principal receives CAN_MANAGE permission on the experiment
- ✅ No manual permission configuration required

**Note:** You may see warnings about "unknown field" during deployment - these can be ignored and will be resolved in the CLI release.

### Additional App Resources

Beyond MLflow experiments, you can grant apps access to other workspace resources by adding them to the `resources` array in databricks.yml:

**Supported resource types:**
- MLflow experiments (experiment tracking)
- SQL warehouses (data querying)
- Model serving endpoints (model inference)
- Unity Catalog volumes (file storage)
- Secrets (secure credential storage)
- Vector search indexes (semantic search)

See: https://docs.databricks.com/aws/en/dev-tools/databricks-apps/resources

### Testing the Deployed App

**1. Generate OAuth Token:**

```bash
# Generate a fresh OAuth token (expires after 1 hour)
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks auth token --host https://your-workspace.cloud.databricks.com
```

Save the `access_token` from the JSON output.

**2. Send Test Request:**

```bash
# Non-streaming request
curl -X POST <app-url>/invocations \
  -H "Authorization: Bearer <oauth-token>" \
  -H "Content-Type: application/json" \
  -d '{"input": [{"role": "user", "content": "hi"}]}'

# Streaming request
curl -X POST <app-url>/invocations \
  -H "Authorization: Bearer <oauth-token>" \
  -H "Content-Type: application/json" \
  -d '{"input": [{"role": "user", "content": "hi"}], "stream": true}'
```

**Note:** OAuth tokens are required for Databricks Apps - PATs are not supported.

### Debugging Deployed Apps

**View app logs:**

```bash
# View recent logs (last 200 lines)
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps logs <app-name>

# Follow logs in real-time
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps logs <app-name> --follow

# Search logs for specific terms
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps logs <app-name> --search "ERROR"

# Filter by source (APP or SYSTEM)
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps logs <app-name> --source APP
```

**Check app status:**

```bash
# Get detailed app status
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps get <app-name> --output json | jq '{app_status, compute_status}'

# List all deployments
DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps list-deployments <app-name>
```

**Common issues:**

1. **502 Bad Gateway** - App container not running or startup failed
   - Check logs: `databricks apps logs <app-name>`
   - Verify app status: `databricks apps get <app-name>`

2. **Authentication Failed** - OAuth token expired or invalid
   - Regenerate token: `databricks auth token`
   - Verify profile is configured with OAuth: `databricks auth login`

3. **App crashed on startup** - Check logs for errors
   - View recent logs: `databricks apps logs <app-name> --tail-lines 100`
   - Look for Python errors, missing dependencies, or configuration issues

### Development Loop

For rapid iteration:

1. **Make code changes locally**
2. **Test locally:** `uv run start-app`
3. **Deploy to Databricks:**
   ```bash
   DATABRICKS_CONFIG_PROFILE=<profile-name> databricks bundle deploy --target dev
   DATABRICKS_CONFIG_PROFILE=<profile-name> databricks apps deploy <app-name> --source-code-path /Workspace/Users/$DATABRICKS_USERNAME/.bundle/<bundle-name>/dev/files
   ```
4. **Test deployed app:**
   ```bash
   TOKEN=$(DATABRICKS_CONFIG_PROFILE=<profile-name> databricks auth token --host <workspace-url> | jq -r '.access_token')
   curl -X POST <app-url>/invocations -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" -d '{"input": [{"role": "user", "content": "test"}]}'
   ```
5. **Check logs:** `databricks apps logs <app-name> --follow`
6. **Iterate**

---

## Agent Framework Capabilities

Reference: https://docs.databricks.com/aws/en/generative-ai/agent-framework/

### Tool Types

1. **Unity Catalog Function Tools** - SQL UDFs managed in UC with built-in governance
2. **Agent Code Tools** - Defined directly in agent code for REST APIs and low-latency operations
3. **MCP Tools** - Interoperable tools via Model Context Protocol (Databricks-managed, external, or self-hosted)

### Built-in Tools

- **system.ai.python_exec** - Execute Python code dynamically within agent queries (code interpreter)

### Common Patterns

- **Structured data retrieval** - Query SQL tables/databases
- **Unstructured data retrieval** - Document search and RAG via Vector Search
- **Code interpreter** - Python execution for analysis via system.ai.python_exec
- **External connections** - Integrate services like Slack via HTTP connections
