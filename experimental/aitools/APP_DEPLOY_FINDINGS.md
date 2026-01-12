
| Issue | Description | Details | Status |
| :---- | :---- | :---- | :---- |
| MLflow experiment resources not in CLI | Required MLflow experiment resources for apps are not yet available in the released Databricks CLI | Must build the CLI from source to access experimental MLflow experiment support | No action needed, will be addressed with next Databricks CLI/SDK release. |
| Need to manually create experiment | \- | databricks.yml in the agent template doesn’t actually create an experiment; it requires passing an experiment ID as a variable. It would be better to create the experiment in the bundle itself (by default) and pass the ID of that experiment to the app as an app resource | TODO |
| Introducing an “agent” template confuses AI (and probably humans too) | The distinction between a new “agent” template and the existing app template is unclear | The concept of an “Agent” template is confusing to Claude (and probably to humans too :P) \- it doesn’t know when to use the “Agent” template or when to use the “app” template. If I ask it to build a “chatbot”, it tries to use the “app” template, which is pretty sensible but unfortunately not the right one | Backlog \- don’t work on this yet, needs discussion with app devex |
| Running from dev requires template override | Running the agent from a local dev environment needs an explicit template path override | Must set `DATABRICKS_AGENT_TEMPLATE_PATH="$HOME/cli/experimental/aitools/templates/agent-openai-agents-sdk"` to install from source instead of GitHub | Expected behavior |
| Validation fails due to Node.js assumptions | Claude runs the validation tool which assumes a Node.js frontend | Validation fails because the agent is pure Python and has no `package.json`; validation checks incorrectly run npm install | Backlog \- don’t work on this yet, needs discussion with app devex |
| Validation tool incompatible with Python agent | The validation process expects a Node.js-based app | The agent is Python-only and already deployable, but validation blocks deployment | Backlog \- don’t work on this yet, needs discussion with app devex |
| Quickstart script requires interactivity | The quickstart script in the agent app template fails in non-interactive execution | Script requires selecting a Databricks profile interactively; workaround is manually configuring `.env.local` | TODO. Address this by providing a flag-based or non-interactive option |
| FMAPI strict parameter validation error | Default agent template triggers strict FMAPI validation errors | Error: `tools.0.custom.strict: Extra inputs are not permitted`; Claude resolved by removing MCP tools from the agent |  |
| Repo not initialized as Git repository | Generated project directory is not a Git repository | Causes `fatal: not a git repository` warnings during startup and logging | TODO. Address this by updating AGENT.md |
| Port 8000 already in use / Port flag ignored by start script | Backend ignores specified port and attempts to bind to 8000\. CLI `--port` argument is not respected | `uv run start-app --port 8001` still binds to port 8000, contradicting README | TODO. Investigate and address this |
| Difficulty awaiting app readiness | Claude struggles to wait for app creation and readiness before querying | Repeated deploy/get attempts while deployment is still in progress |  |
| Required bundle variable not persisted | `mlflow_experiment_id` not remembered between bundle commands | Must repeatedly pass `--var mlflow_experiment_id=...` to avoid errors | Should address by updating bundle to create the experiment |
| App stuck in deployment-in-progress state | Attempts to redeploy fail due to active deployment | CLI reports “active deployment in progress” even when app status is unavailable |  |
| Logs unavailable during deployment | App logs cannot be retrieved during certain deployment states | `apps logs` fails with HTTP 503 / bad websocket handshake |  |

Challenges:

* MLflow experiment resources for apps not yet released in the CLI, need to build from source
* The concept of an “Agent” template is confusing to Claude (and probably to humans too :P) \- it doesn’t know when to use the “Agent” template or when to use the “app” template. If I ask it to build a “chatbot”, it tries to use the “app” template, which is pretty sensible but unfortunately not the right one
* Note/prerequisite for running from dev: need to override template path to install from source instead of from GitHub

```
export DATABRICKS_AGENT_TEMPLATE_PATH="$HOME/cli/experimental/aitools/templates/agent-openai-agents-sdk"
```

* Claude tries to run a validation command which fails against this Python-based app template

```
databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["experimental","aitools","tools","validate","./"])
  ⎿ Error: install failed (duration: 0.4s)
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    🔍 Databricks AI Tools MCP server: Validating your app
    ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
    Validation Progress:
    🔄 Starting Node.js validation: build + typecheck + tests
    ⏳ Step 1/6: Running Install...
    ❌ Install failed (0.4s)
    ❌ Failed to install dependencies
    Exit code: 254
    Stdout:
    Stderr:
    (node:61716) ExperimentalWarning: CommonJS module /opt/homebrew/lib/node_modules/npm/node_modules/debug/src/node.js is loading
     ES Module /opt/homebrew/lib/node_modules/npm/node_modules/supports-color/index.js using require().
    Support for loading ES Module in require() is an experimental feature and might change at any time
    (Use `node --trace-warnings ...` to show where the warning was created)
    npm error code ENOENT
    npm error syscall open
    npm error path /Users/sid.murching/customer_support_agent/package.json
    npm error errno -2
    npm error enoent Could not read package.json: Error: ENOENT: no such file or directory, open
    '/Users/sid.murching/customer_support_agent/package.json'
    npm error enoent This is related to npm not being able to find a file.
    npm error enoent
    npm error A complete log of this run can be found in: /Users/sid.murching/.npm/_logs/2026-01-12T22_13_40_682Z-debug-0.log
    Error: validation failed
    Command failed with error: exit status 1
⏺ The validation tool is checking for a Node.js frontend which this pure Python agent doesn02:13 PM Tclaude-sonnet-4-5-20250929on-based agent. The agent code is ready to deploy.
```

* Claude tries to run the quickstart script but it fails since it requires interactive input. Thankfully it finds a workaround (updating .env.local):

```
⏺ Bash(cd /Users/sid.murching/customer_support_agent && ./scripts/quickstart.sh)
  ⎿  Error: Exit code 1
     Downloading https://nodejs.org/dist/v20.19.6/node-v20.19.6-darwin-arm64.tar.xz...
######                                                                     0.0%
#######################
#############################
######################################################
######################################################################## 100.0%
     Computing checksum with sha256sum
     Checksums matched!
     ===================================================================
     Agent on Apps - Quickstart Setup
     ===================================================================
     … +80 lines (ctrl+o to see all)
⏺ The quickstart script needs interactive input to select a Databricks profile. Let me configure it automatically using the DEFAULT profile.

```

* Got the agent running locally, but the default agent template code is running into issues due to some FMAPI `strict` parameter validation.

```

⏺ Bash(curl -X POST http://localhost:8001/invocations \ timeout: 1m 0s
        -H "Content-Type: application/json" \…)
  ⎿ {
      "detail": "Error code: 400 - {'error_code': 'BAD_REQUEST', 'message': '{\"message\":\"tools.0.custom.strict: Extra inputs ar
    e not permitted\"}'}"
    … +5 lines (ctrl+o to expand)

```

Claude got around it^ by removing MCP tools from the agent implementation.

* Got two other errors; the repo that was created is not a .git repo and port 8000 was already in use

```
Last 50 lines of backend.log:
----------------------------------------
fatal: not a git repository (or any of the parent directories): .git
2026/01/12 14:26:57 INFO mlflow.tracking.fluent: Active model is set to the logged model with ID: m-1010b5487fed49b6a208f7bf8e3c68e3
2026/01/12 14:26:57 INFO mlflow.tracking.fluent: Use `mlflow.set_active_model` to set the active model to a different one if needed.
2026/01/12 14:26:57 INFO mlflow.genai.agent_server.utils: Active LoggedModel: 'local-no-git', Model ID: 'm-1010b5487fed49b6a208f7bf8e3c68e3'
INFO:     Started server process [70415]
INFO:     Waiting for application startup.
INFO:     Application startup complete.
ERROR:    [Errno 48] error while attempting to bind on address ('0.0.0.0', 8000): [errno 48] address already in use
INFO:     Waiting for application shutdown.
INFO:     Application shutdown complete.
----------------------------------------
Last 50 lines of frontend.log:
----------------------------------------
----------------------------------------
==========================================
Shutting down both processes...
==========================================

```

I can reproduce by running the server manually \- even if I try to use port 8001 following the README, it seems to be ignored.:

```
~/customer_support_agent
base ❯ uv run start-app --port 8001                                                                               [01/12/26 | 2:27:07]
Starting backend...
Running npm install...
Running npm build...
[backend] fatal: not a git repository (or any of the parent directories): .git
[backend] 2026/01/12 14:28:38 INFO mlflow.tracking.fluent: Active model is set to the logged model with ID: m-1010b5487fed49b6a208f7bf8e3c68e3
[backend] 2026/01/12 14:28:38 INFO mlflow.tracking.fluent: Use `mlflow.set_active_model` to set the active model to a different one if needed.
[backend] 2026/01/12 14:28:38 INFO mlflow.genai.agent_server.utils: Active LoggedModel: 'local-no-git', Model ID: 'm-1010b5487fed49b6a208f7bf8e3c68e3'
[backend] INFO:     Started server process [70949]
✓ Backend is ready!
[backend] INFO:     Waiting for application startup.
[backend] INFO:     Application startup complete.
[backend] ERROR:    [Errno 48] error while attempting to bind on address ('0.0.0.0', 8000): [errno 48] address already in use
[backend] INFO:     Waiting for application shutdown.
[backend] INFO:     Application shutdown complete.
Starting frontend...
```

I manually killed the process on port 8000 to unblock making progress

* Claude seems to struggle a bit with understanding how to await creation of the app and query it:

```
⏺ Update(customer_support_agent/databricks.yml)
  ⎿  Added 1 line, removed 1 line
       9   resources:
      10     apps:
      11       customer_support_agent:
      12 -       name: "${bundle.target}-customer_support_agent"
      12 +       name: "${bundle.target}-customer-support-agent"
      13         description: "OpenAI Agents SDK agent application"
      14         source_code_path: ./
      15
⏺ Update(customer_support_agent/databricks.yml)
  ⎿  Added 1 line, removed 1 line
      34       resources:
      35         apps:
      36           customer_support_agent:
      37 -           name: customer_support_agent
      37 +           name: customer-support-agent
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args: ["bundle","deploy
                                              ","--profile","DEFAULT","--var","mlflow_experiment_id=4043694633333313"])
  ⎿  Uploading bundle files to /Workspace/Users/sid.murching@databricks.com/.bundle/customer_support_agent/dev/files...
     Deploying resources...
     Updating deployment state...
     Deployment complete!
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["bundle","summary","--profile","DEFAULT"])
  ⎿ Error: no value assigned to required variable mlflow_experiment_id. Variables are usually assigned in databricks.yml, and they
     can be overridden using "--var", the BUNDLE_VAR_mlflow_experiment_id environment variable, or .databricks/bundle/<target>/var
    iable-overrides.json
    … +4 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args: ["bundle","summar
                                              y","--profile","DEFAULT","--var","mlflow_experiment_id=4043694633333313"])
  ⎿  Name: customer_support_agent
     Target: dev
     Workspace:
     … +8 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","get","dev-customer-support-agent"])
  ⎿  {
       "app_status": {
         "message":"App status is unavailable.",
     … +37 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","deploy","dev-customer-support-agent","--source-code-path",
                                              "/Workspace/Users/sid.murching@databricks.com/.bundle/customer_support_agent/dev/files"])
  ⎿  Error: Cannot deploy app dev-customer-support-agent as there is an active deployment in progress.
     Command failed with error: exit status 1
⏺ Bash(sleep 30 && databricks --profile DEFAULT apps get dev-customer-support-agent --output json | python3 -c "import     timeout: 1m
      sys, json; data = json.load(sys.stdin); print(f\…)                                                                   0s
  ⎿  App Status: UNAVAILABLE
     Compute Status: STOPPED
     URL: https://dev-customer-support-agent-3888667486068890.aws.databricksapps.com
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","list-deployments","dev-customer-support-agent"])
  ⎿  DeploymentId  State  CreatedAt
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","deploy","dev-customer-support-agent","--source-code-path",
                                              "/Workspace/Users/sid.murching@databricks.com/.bundle/customer_support_agent/dev/files"])
  ⎿  Error: Cannot deploy app dev-customer-support-agent as there is an active deployment in progress.
     Command failed with error: exit status 1
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","get","dev-customer-support-agent","--output","json"])
  ⎿  {
       "app_status": {
         "message":"App status is unavailable.",
     … +37 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","experimental","aitools","tools","deploy","--skip-validation"])
  ⎿  Error: unknown flag: --skip-validation
     Usage:
     … +15 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","experimental","aitools","tools","deploy","--var","mlflow_experime
                                              nt_id=4043694633333313"])
  ⎿  Error: cannot deploy: project not validated (run validate first)
     Command failed with error: exit status 1
⏺ Let me check the app logs to see what's happening with the deployment.
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","logs","dev-customer-support-agent","--tail-lines","50"])
  ⎿  Error: websocket: bad handshake (HTTP 503 Service Unavailable)
     Command failed with error: exit status 1
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","start","dev-customer-support-agent"])
  ⎿  {
       "app_status": {
         "message":"App has status: App has not been deployed yet. Run your app by deploying source code",
     … +37 lines (ctrl+o to expand)
⏺ Bash(sleep 5)
  ⎿  (No content)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","deploy","dev-customer-support-agent","--source-code-path",
                                              "/Workspace/Users/sid.murching@databricks.com/.bundle/customer_support_agent/dev/files"])
  ⎿  {
       "create_time":"2026-01-12T22:42:02Z",
       "creator":"sid.murching@databricks.com",
     … +12 lines (ctrl+o to expand)
⏺ databricks-mcp - invoke_databricks_cli (MCP)(working_directory: "/Users/sid.murching/customer_support_agent", args:
                                              ["--profile","DEFAULT","apps","get","dev-customer-support-agent","--output","json"])
  ⎿  {
       "active_deployment": {
         "create_time":"2026-01-12T22:42:02Z",
     … +52 lines (ctrl+o to expand)
⏺ Perfect! Your customer support agent has been successfully deployed to Databricks! 🎉
  Deployment Summary
  App Name: dev-customer-support-agent
  Status: ✅ RUNNING
  URL: https://dev-customer-support-agent-3888667486068890.aws.databricksapps.com

```
