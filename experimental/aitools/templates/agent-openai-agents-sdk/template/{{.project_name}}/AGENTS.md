# Agent OpenAI Agents SDK Development Guide

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

**Create app:**

```bash
databricks apps create agent-openai-agents-sdk
```

**Sync files:**

```bash
DATABRICKS_USERNAME=$(databricks current-user me | jq -r .userName)
databricks sync . "/Users/$DATABRICKS_USERNAME/agent-openai-agents-sdk"
```

**Deploy:**

```bash
databricks apps deploy agent-openai-agents-sdk --source-code-path /Workspace/Users/$DATABRICKS_USERNAME/agent-openai-agents-sdk
```

**Query deployed app:**

Generate OAuth token (PATs are not supported):

```bash
databricks auth token
```

Send request:

```bash
curl -X POST <app-url>/invocations \
  -H "Authorization: Bearer <oauth-token>" \
  -H "Content-Type: application/json" \
  -d '{ "input": [{ "role": "user", "content": "hi" }], "stream": true }'
```

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
