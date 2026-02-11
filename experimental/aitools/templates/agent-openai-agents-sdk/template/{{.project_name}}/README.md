# Responses API Agent

This template defines a conversational agent app. The app comes with a built-in chat UI, but also exposes an API endpoint for invoking the agent so that you can serve your UI elsewhere (e.g. on your website or in a mobile app).

The agent in this template implements the [OpenAI Responses API](https://platform.openai.com/docs/api-reference/responses) interface. It has access to a single tool; the [built-in code interpreter tool](https://docs.databricks.com/aws/en/generative-ai/agent-framework/code-interpreter-tools#built-in-python-executor-tool) (`system.ai.python_exec`) on Databricks. You can customize agent code and test it via the API or UI.

The agent input and output format are defined by MLflow's ResponsesAgent interface, which closely follows the [OpenAI Responses API](https://platform.openai.com/docs/api-reference/responses) interface. See [the MLflow docs](https://mlflow.org/docs/latest/genai/flavors/responses-agent-intro/) for input and output formats for streaming and non-streaming requests, tracing requirements, and other agent authoring details.

## Quick start

Run the `./scripts/quickstart.sh` script to quickly set up your local environment and start the agent server. At any step, if there are issues, refer to the manual local development loop setup below.

This script will:

1. Verify uv, nvm, and Databricks CLI installations
2. Configure Databricks authentication
3. Configure agent tracing, by creating and linking an MLflow experiment to your app
4. Start the agent server and chat app

```bash
./scripts/quickstart.sh
```

After the setup is complete, you can start the agent server and the chat app locally with:

```bash
uv run start-app
```

This will start the agent server and the chat app at http://localhost:8000.

**Next steps**: see [modifying your agent](#modifying-your-agent) to customize and iterate on the agent code.

## Manual local development loop setup

1. **Set up your local environment**
   Install `uv` (python package manager), `nvm` (node version manager), and the Databricks CLI:

   - [`uv` installation docs](https://docs.astral.sh/uv/getting-started/installation/)
   - [`nvm` installation](https://github.com/nvm-sh/nvm?tab=readme-ov-file#installing-and-updating)
     - Run the following to use Node 20 LTS:
       ```bash
       nvm use 20
       ```
   - [`databricks CLI` installation](https://docs.databricks.com/aws/en/dev-tools/cli/install)

2. **Set up local authentication to Databricks**

   In order to access Databricks resources from your local machine while developing your agent, you need to authenticate with Databricks. Choose one of the following options:

   **Option 1: OAuth via Databricks CLI (Recommended)**

   Authenticate with Databricks using the CLI. See the [CLI OAuth documentation](https://docs.databricks.com/aws/en/dev-tools/cli/authentication#oauth-user-to-machine-u2m-authentication).

   ```bash
   databricks auth login
   ```

   Set the `DATABRICKS_CONFIG_PROFILE` environment variable in your .env.local file to the profile you used to authenticate:

   ```bash
   DATABRICKS_CONFIG_PROFILE="DEFAULT" # change to the profile name you chose
   ```

   **Option 2: Personal Access Token (PAT)**

   See the [PAT documentation](https://docs.databricks.com/aws/en/dev-tools/auth/pat#databricks-personal-access-tokens-for-workspace-users).

   ```bash
   # Add these to your .env.local file
   DATABRICKS_HOST="https://host.databricks.com"
   DATABRICKS_TOKEN="dapi_token"
   ```

   See the [Databricks SDK authentication docs](https://docs.databricks.com/aws/en/dev-tools/sdk-python#authenticate-the-databricks-sdk-for-python-with-your-databricks-account-or-workspace).

3. **Create and link an MLflow experiment to your app**

   Create an MLflow experiment to enable tracing and version tracking. This is automatically done by the `./scripts/quickstart.sh` script.

   Create the MLflow experiment via the CLI:

   ```bash
   DATABRICKS_USERNAME=$(databricks current-user me | jq -r .userName)
   databricks experiments create-experiment /Users/$DATABRICKS_USERNAME/agents-on-apps
   ```

   Make a copy of `.env.example` to `.env.local` and update the `MLFLOW_EXPERIMENT_ID` in your `.env.local` file with the experiment ID you created. The `.env.local` file will be automatically loaded when starting the server.

   ```bash
   cp .env.example .env.local
   # Edit .env.local and fill in your experiment ID
   ```

   See the [MLflow experiments documentation](https://docs.databricks.com/aws/en/mlflow/experiments#create-experiment-from-the-workspace).

4. **Test your agent locally**

   Start up the agent server and chat UI locally:

   ```bash
   uv run start-app
   ```

   Query your agent via the UI (http://localhost:8000) or REST API:

   **Advanced server options:**

   ```bash
   uv run start-server --reload   # hot-reload the server on code changes
   uv run start-server --port 8001 # change the port the server listens on
   uv run start-server --workers 4 # run the server with multiple workers
   ```

   - Example streaming request:
     ```bash
     curl -X POST http://localhost:8000/invocations \
     -H "Content-Type: application/json" \
     -d '{ "input": [{ "role": "user", "content": "hi" }], "stream": true }'
     ```
   - Example non-streaming request:
     ```bash
     curl -X POST http://localhost:8000/invocations  \
     -H "Content-Type: application/json" \
     -d '{ "input": [{ "role": "user", "content": "hi" }] }'
     ```

## Modifying your agent

See the [OpenAI Agents SDK documentation](https://platform.openai.com/docs/guides/agents-sdk) for more information on how to edit your own agent.

Required files for hosting with MLflow `AgentServer`:

- `agent.py`: Contains your agent logic. Modify this file to create your custom agent. For example, you can [add agent tools](https://docs.databricks.com/aws/en/generative-ai/agent-framework/agent-tool) to give your agent additional capabilities
- `start_server.py`: Initializes and runs the MLflow `AgentServer` with agent_type="ResponsesAgent". You don't have to modify this file for most common use cases, but can add additional server routes (e.g. a `/metrics` endpoint) here

**Common customization questions:**

**Q: Can I add additional files or folders to my agent?**
Yes. Add additional files or folders as needed. Ensure the script within `pyproject.toml` runs the correct script that starts the server and sets up MLflow tracing.

**Q: How do I add dependencies to my agent?**
Run `uv add <package_name>` (e.g., `uv add "mlflow-skinny[databricks]"`). See the [python pyproject.toml guide](https://packaging.python.org/en/latest/guides/writing-pyproject-toml/#dependencies-and-requirements).

**Q: Can I add custom tracing beyond the built-in tracing?**
Yes. This template uses MLflow's agent server, which comes with automatic tracing for agent logic decorated with `@invoke()` and `@stream()`. It also uses [MLflow autologging APIs](https://mlflow.org/docs/latest/genai/tracing/#one-line-auto-tracing-integrations) to capture traces from LLM invocations. However, you can add additional instrumentation to capture more granular trace information when your agent runs. See the [MLflow tracing documentation](https://docs.databricks.com/aws/en/mlflow3/genai/tracing/app-instrumentation/).

**Q: How can I extend this example with additional tools and capabilities?**
This template can be extended by integrating additional MCP servers, Vector Search Indexes, UC Functions, and other Databricks tools. See the ["Agent Framework Tools Documentation"](https://docs.databricks.com/aws/en/generative-ai/agent-framework/agent-tool).

## Evaluating your agent

Evaluate your agent by calling the invoke function you defined for the agent locally.

- Update your `evaluate_agent.py` file with the preferred evaluation dataset and scorers.

Run the evaluation using the evaluation script:

```bash
uv run agent-evaluate
```

After it completes, open the MLflow UI link for your experiment to inspect results.

## Deploying to Databricks Apps

0. **Create a Databricks App**:
   Ensure you have the [Databricks CLI](https://docs.databricks.com/aws/en/dev-tools/cli/tutorial) installed and configured.

   ```bash
   databricks apps create agent-openai-agents-sdk
   ```

1. **Set up authentication to Databricks resources**

   For this example, you need to add an MLflow Experiment as a resource to your app. Grant the App's Service Principal (SP) permission to edit the experiment by clicking `edit` on your app home page. See the [Databricks Apps MLflow experiment documentation](https://docs.databricks.com/aws/en/dev-tools/databricks-apps/mlflow) for more information.

   To grant access to other resources like serving endpoints, genie spaces, UC Functions, and Vector Search Indexes, click `edit` on your app home page to grant the App's SP permission. See the [Databricks Apps resources documentation](https://docs.databricks.com/aws/en/dev-tools/databricks-apps/resources).

   For resources that are not supported yet, see the [Agent Framework authentication documentation](https://docs.databricks.com/aws/en/generative-ai/agent-framework/deploy-agent#automatic-authentication-passthrough) for the correct permission level to grant to your app SP.

   **On-behalf-of (OBO) User Authentication**: Use `get_user_workspace_client()` from `agent_server.utils` to authenticate as the requesting user instead of the app service principal. See the [OBO authentication documentation](https://docs.databricks.com/aws/en/dev-tools/databricks-apps/auth?language=Streamlit#retrieve-user-authorization-credentials).

2. **Sync local files to your workspace**

   See the [Databricks Apps deploy documentation](https://docs.databricks.com/aws/en/dev-tools/databricks-apps/deploy?language=Databricks+CLI#deploy-the-app).

   ```bash
   DATABRICKS_USERNAME=$(databricks current-user me | jq -r .userName)
   databricks sync . "/Users/$DATABRICKS_USERNAME/agent-openai-agents-sdk"
   ```

3. **Deploy your Databricks App**

   See the [Databricks Apps deploy documentation](https://docs.databricks.com/aws/en/dev-tools/databricks-apps/deploy?language=Databricks+CLI#deploy-the-app).

   ```bash
   databricks apps deploy agent-openai-agents-sdk --source-code-path /Workspace/Users/$DATABRICKS_USERNAME/agent-openai-agents-sdk
   ```

4. **Query your agent hosted on Databricks Apps**

   Databricks Apps are _only_ queryable via OAuth token. You cannot use a PAT to query your agent. Generate an [OAuth token with your credentials using the Databricks CLI](https://docs.databricks.com/aws/en/dev-tools/cli/authentication#u2m-auth):

   ```bash
   databricks auth login --host <https://host.databricks.com>
   databricks auth token
   ```

   Send a request to the `/invocations` endpoint:

   - Example streaming request:

     ```bash
     curl -X POST <app-url.databricksapps.com>/invocations \
        -H "Authorization: Bearer <oauth token>" \
        -H "Content-Type: application/json" \
        -d '{ "input": [{ "role": "user", "content": "hi" }], "stream": true }'
     ```

   - Example non-streaming request:

     ```bash
     curl -X POST <app-url.databricksapps.com>/invocations \
        -H "Authorization: Bearer <oauth token>" \
        -H "Content-Type: application/json" \
        -d '{ "input": [{ "role": "user", "content": "hi" }] }'
     ```

For future updates to the agent, sync and redeploy your agent.

### FAQ

- For a streaming response, I see a 200 OK in the logs, but an error in the actual stream. What's going on?
  - This is expected behavior. The initial 200 OK confirms stream setup; streaming errors don't affect this status.
- When querying my agent, I get a 302 error. What's going on?
  - Use an OAuth token. PATs are not supported for querying agents.
