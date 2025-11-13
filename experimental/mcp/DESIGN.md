Add a 'databricks experimental mcp' command that behaves as follows:

* databricks experimental mcp --help: shows help
* databricks experimental mcp: shows help (like other command groups)
* databricks experimental mcp install: installs the server in coding agents
* databricks experimental mcp server: starts the mcp server (subcommand)
* databricks experimental mcp uninstall: uninstalls the server from coding agents (subcommand - not implemented; errors out and tells the user to ask their local coding agent to uninstall the Databricks CLI MCP server)
* databricks experimental mcp tool <tool_name> --json <json>: runs a specific MCP tool for acceptance testing (hidden subcommand)

non-functional requirements:
- any errors that these commands give should be friendly, concise, actionable.
- this code needs to be modular (e.g cursor installation/detection is one module) and needs to have unit tests
- write code docs and documentation in a very concise and minimal way, and keep maintainers in mind; look at other
  modules for inspiration
- take AGENTS.md into account when building this
- MANDATORY: never invoke the databricks cli directly, instead use the invoke_databricks_cli tool!
- all AI-facing prompts and guidance text must be stored in .tmpl files in the tools/prompts/ directory (not hardcoded in Go files)
- Resource-specific code is modularized in cmd/mcp/tools/resources/ directory. Each resource type (app, job, pipeline, dashboard) has its own file (apps.go, jobs.go, etc.) that implements:
  - Add* function: handles adding the resource to a project
  - AnalyzeGuidance* function: returns resource-specific guidance for the AI agent
  This structure makes it easy for teams (e.g., the apps team) to customize how the agent works with their specific resource type by editing the corresponding resource file.
- MANDATORY: you need to experiment locally with the claude cli and cursor cli to make sure this actually works as expected.
  The testing approach should be:

  Example test command:
  ```bash
  rm -rf /tmp/blank; mkdir -p /tmp/blank; cd /tmp/blank;
  claude --allow-dangerously-skip-permissions "Create a new Databricks app that shows a dashboard with taxi trip fares per city, then preview it and open it in my browser. If the databricks-mcp MCP fails, stop immediately and ask for my guidance."
  ```

  You should test multiple scenarios:
  - Creating a project with a simple job that lists taxis and running it
  - Creating an app with a dashboard
  - Adding resources to an existing project
  - The key is to use the Claude CLI to issue prompts as a user would, NOT to directly call MCP tools yourself
  - If the MCP server has issues, Claude Code should surface clear error messages

  This is the most important part of the work; i expect you to deeply experiment with the new mcp server; add it, try it, remove it, add it again, use it to build things that can run.
- MANDATORY: at the very end, compare what you built again to the instructions in this doc; go over each point, does it work, is it complete?


To illustrate how the install command works:

```
$ databricks experimental mcp install

  ▄▄▄▄▄▄▄▄   Databricks CLI
  ██▌  ▐██   MCP Server
  ▀▀▀▀▀▀▀▀

╔════════════════════════════════════════════════════════════════╗
║  ⚠️  EXPERIMENTAL: This command may change in future versions  ║
╚════════════════════════════════════════════════════════════════╝

Which coding agents would you like to install the MCP server for?

Install for Claude Code? [Use arrows to move]
> yes
  no

Install for Cursor? [Use arrows to move]
> yes
  no

Show manual installation instructions for other agents? [Use arrows to move]
> no
  yes

Installing MCP server for Claude Code...
✓ Installed for Claude Code

Installing MCP server for Cursor...
✓ Installed for Cursor

You can now use your coding agent to interact with Databricks.
Try asking: 'Create a new Databricks project with a job or an app'
```

Implementation notes:
- Uses cmdio.AskSelect (arrow keys, all default to "yes" for easy Enter×3)
- Always prompts for both Claude Code and Cursor (no detection filtering)
- Gracefully skips missing agents with yellow warning instead of erroring
- Line replacement: "Installing..." → "✓ Installed" (or "⊘ Skipped") using \r

Now databricks experimental mcp server should actually start an MCP server that we actually use to describe
the system as a whole a bit (btw each tool should be defined in a separate .go file, right!):
- when starting up it should do a the 'roots/list' to the agent.
  - if that doesn't work or if there is more than one root => error out
  - look at the root path. if there is already a databricks.yml file, that means the user already initialized a project; keep track of that.

- for the tools below, there is a common initialization step. you need to check if you're
  properly authenticated to the workspace. you can do so by using the invoke_databricks_cli tool
  to run 'jobs get <random id>' (pick any random id like 123456789).
  if you get an authentication error, the tools need to return an error saying that they first need
  to log in to databricks. to do so they need to use the invoke_databricks_cli tool to run:
  'auth login --profile DEFAULT --host <my company url, e.g. mycompany.databricks.com>'.
  the AI needs to ask the user for this url, it cannot guess it.
  once logged in, the tools will work! the AI should also point to https://docs.databricks.com/getting-started/free-edition as a page where users can setup their own fully free account for experimentation.

- the "invoke_databricks_cli" tool:
    - description: run any databricks CLI command. this is a passthrough to the databricks CLI.
      use this tool whenever you need to run databricks CLI commands like 'databricks bundle deploy',
      'databricks bundle validate', 'databricks bundle run', 'databricks auth login', etc.
      the reason this tool exists (instead of invoking the databricks CLI directly) is to make it
      easier for users to allow-list commands compared to allow-listing shell commands.
    - parameter: command - the full databricks CLI command to run, e.g. "bundle deploy" or "bundle validate"
      (note: do not include the "databricks" prefix in the command parameter)
    - parameter: working_directory - optional. the directory to run the command in. defaults to the current directory.
    - output: the stdout and stderr from the command, plus the exit code
    - implementation guidance: this should just invoke the databricks CLI and return the output.
      make sure to properly handle the working directory if provided.
    - further implementation guidance: i want an acceptance test for this command. it should just call the 'help' command.

- the "init_project" tool:
    - description: initializes a new databricks project structure. Use this to create a new project. After initialization, use add_project_resource to add resources such as apps, jobs, dashboards, pipelines, etc.
    - parameter: project_name - a name for this project in snake_case; ask the user about this if it's not clear from the context
    - parameter: project_path - a fully qualified path for the directory to create the new project in. Usually this should be in the current directory! But if it already has a lot of other things then it should be a subdirectory.
    - action to perform when this runs: use the invoke_databricks_cli tool to run
      'bundle init default-minimal --config-file /tmp/...' where you set the 'project_name' and other
      parameters. use personal schemas and the default catalog.
      note that default-minimal creates a subdirectory called 'project_name'! this is not needed. just move everything
      (recursively) in that subdirectory to the target directory from project_path.
      after initialization, creates a CLAUDE.md file (if the calling MCP client is Claude Code) or AGENTS.md file (otherwise)
      with project-specific agent instructions. The file includes:
      - Installation instructions for the Databricks CLI MCP server (if not yet installed)
      - Guidance to use the mcp__databricks-mcp__analyze_project tool when opening the project
      The client is detected at runtime from the MCP initialize request's clientInfo field.
    - guidance on how to implement this: do some trial and error to make the init command work.
      do a non-forward merge of origin/add-default-minimal to get the minimal template!
    - output: returns a success message with a WARNING that this is an EMPTY project with NO resources, and that add_project_resource MUST be called if the user asked for a specific resource. followed by the same guidance as the analyze_project tool (calls analyze_project internally)
    - further implementation guidance: i want an acceptance test for this command. it should lead to a project
      that can pass a 'bundle validate' command!

- the "analyze_project" tool:
    - description: REQUIRED FIRST STEP: If databricks.yml exists in the directory, you MUST call this tool before using Read, Glob, or any other tools. Databricks projects require specialized commands that differ from standard Python/Node.js workflows - attempting standard approaches will fail. This tool is fast and provides the correct commands for preview/deploy/run operations.
    - parameter: project_path - a fully qualified path of the project to operate on. <if we determined there is a project in /: "by default, the current directory", if not: this must be a directory with a databricks.yml file>
    - output:
      - summary: contents of the 'bundle summary' command run in this dir using the invoke_databricks_cli tool.
        <implementation guidance: you need to run this command in the mcp! if it fails, just include the failure output>
      - guidance:
          - "Below is guidance for how to work with this project.
             - IMPORTANT: you want to give the user some idea of how to get started; you can suggest
               prompts such as "Create an app that shows a chart with taxi fares by city"
               or "Create a job that summarizes all taxi data using a notebook"
             - IMPORTANT: Most interactions are done with the Databricks CLI. YOU (the AI) must use the invoke_databricks_cli tool to run commands - never suggest the user runs CLI commands directly!
             - IMPORTANT: to add new resources to a project, use the 'add_project_resource' mcp tool.
             - MANDATORY: always deploy with invoke_databricks_cli 'bundle deploy', never with 'apps deploy'
             - Note that Databricks resources are defined in resources/*.yml files. See https://docs.databricks.com/dev-tools/bundles/settings for a reference!

          - Common guidance about getting started and using the CLI (should draw inspiration from the original default_python_template_readme.md file, extracting common instructions that are not app-specific)
          - <contents of the <project-dir>/README.md (if it exists). This provides project-specific guidance that complements the common guidance>

- the "add_project_resource" tool:
   - description: extend the current project with a new app, job, pipeline, or dashboard
   - parameter: type - app, job, pipeline, or dashboard
   - parameter: name - the name of the new resource (for example: new_app); should not exist yet in resources/
   - parameter: template - optional. only fill this in when asked.
   - implementation guidance:
     - (i have some idea how to implement apps, as described below, but for now just error say its not implemented)
       for apps, there are templates in https://github.com/databricks/app-templates.
       - if no template was given,  error out and tell the AI: either pick a template from the list of templates, or let the user pick. if the user didn't pick a template but did describe an app then just
       use nodejs-fastapi-hello-world-app as a starting point. <implementation guidance: you need
       to do a shallow clone of https://github.com/databricks/app-templates to get this list of template names!>
      - if a template _was_ given then you should create a shallow git clone of https://github.com/databricks/app-templates in /tmp and then copy one of the template dirs (e.g. e2e-chatbot-app-next) to a folder with that name (e.g. e2e-chatbot-app-next). you should also create an associated resources/*.yml (e2e-chatbot-app-next.yml) see https://github.com/databricks/bundle-examples/blob/main/knowledge_base/databricks_app/resources/hello_world.job.yml for a starting point.
    - for jobs, the template parameter needs to be sql or python. error out if not specified; the ai needs to
      ask what language the user wants if this was not clear from the context.
      if a template is specified then do a shallow clone of https://github.com/databricks/bundle-examples,
      and take default_python or default_sql as a starting point depending on the language.
      you need to copy resources/*.job.yml but rename them to resources/<name>.job.yml
      for python, you need to copy src/default_python (but rename to src/<name>) and src/tests
      for sql, you need to copy src/*.sql (dont overwrite anything)
    - for pipelines, most of the guidance is the same (the implementation could be shared?):
      the template parameter needs to be sql or python. error out if not specified; the ai needs to
      ask what language the user wants if this was not clear from the context.
      if a template is specified then do a shallow clone of https://github.com/databricks/bundle-examples,
      and take lakeflow_pipelines_python or lakeflow_pipelines_sql as a starting point depending on the language.
      you need to copy resources/*.pipeline.yml but rename them to resources/<name>.pipeline.yml
      copy src/lakeflow_pipeline_*/** but rename to src/<name>/**
    - for dashboards, do a shallow clone of https://github.com/databricks/bundle-examples.
      use knowledge_base/dashboard_nyc_taxi as a starting point.
      you need to copy resources/*.dashboard.yml but rename them to resources/<name>.dashboard.yml
      copy src/*.lvdash.json but rename to src/<name>.lvdash.json
    - note that all of the above (apps, jobs, pipelines, dashboards) should include a note that says "FIXME: this should rely on the databricks bundle generate command"
  - output: if any of the resource types from above (e.g. a python job) were instantiated,
            the output needs to respond with guidance yet. it should say that the MCP only
            created a starting point and that the AI needs to iterate over it:
            1. use the analyze_project tool to learn about the current project structure and how to use it
            2. validate that the extensions are correct using the invoke_databricks_cli tool to run 'bundle validate'. for apps, also check that any warehouse references in the resource/*.yml file are valid.
            3. should fix any errors, and eventually should deploy to dev using the invoke_databricks_cli tool to run 'bundle deploy --target dev'
            for a pipeline, it can also use the invoke_databricks_cli tool to run 'bundle run <pipeline_name> --validate-only' to
            validate that the pipeline is correct.
  - further implementation guidance: i want acceptance tests for each of these project types (app, dashboard, job, pipeline);
    this means they should be exposed as a hidden command like 'databricks experimental mcp tool add_project_resource --json <json>'. having these tests will be instrumental for iterating on them; the initing should not fail! note that the tool subcommand should just assume that the cwd is the current project dir.

- the "explore" tool:
    - description: CALL THIS FIRST when user mentions a workspace by name or asks about workspace resources. Shows available workspaces/profiles, default warehouse, and provides guidance on exploring jobs, clusters, catalogs, and other Databricks resources. Use this to discover what's available before running CLI commands.
    - no parameters needed
    - implementation:
      - Determines a default SQL warehouse for queries using GetDefaultWarehouse():
        1. Lists all warehouses using 'databricks warehouses list --output json'
        2. Prefers RUNNING warehouses (pick first one found)
        3. If none running, picks first STOPPED warehouse (warehouses auto-start when queried)
        4. If no warehouses available, returns error directing user to create one
      - Shows workspace/profile information:
        1. Reads available profiles from ~/.databrickscfg using libs/databrickscfg/profile package
        2. Shows current profile (from DATABRICKS_CONFIG_PROFILE env var or DEFAULT)
        3. Lists all available workspaces with their host URLs and cloud providers
        4. Provides guidance on using --profile flag to switch workspaces:
           - Example: invoke_databricks_cli '--profile prod catalogs list'
        5. Only shows profile list if multiple profiles exist (saves tokens for single-profile setups)
      - Returns concise guidance text that explains:
        1. Current workspace profile and host
        2. Available workspace profiles (if multiple exist)
        3. The warehouse ID that can be used for queries
        4. How to execute SQL queries using Statement Execution API:
           - invoke_databricks_cli 'api post /api/2.0/sql/statements --json {"warehouse_id":"...","statement":"SELECT ...","wait_timeout":"30s"}'
           - Mentions using the warehouse ID shown above
        5. How to explore workspace resources:
           - Jobs: invoke_databricks_cli 'jobs list', 'jobs get <job_id>'
           - Clusters: invoke_databricks_cli 'clusters list', 'clusters get <cluster_id>'
           - Unity Catalog: invoke_databricks_cli 'catalogs list', 'schemas list', 'tables list', 'tables get'
           - Workspace files: invoke_databricks_cli 'workspace list <path>'
        6. Reminder to use --profile flag for non-default workspaces
      - Key design: Single concise endpoint that provides guidance, not many separate tools
    - output: Guidance text with workspace/profile info, warehouse info, and commands for exploring jobs, clusters, data, and other resources
    - implementation: Single explore.go file with GetDefaultWarehouse, getCurrentProfile, and getAvailableProfiles helpers
    - key use case: When user asks about a specific workspace (e.g., "what jobs do I have in my dogfood workspace"), agent should call this FIRST to see available workspaces and get the correct profile name
