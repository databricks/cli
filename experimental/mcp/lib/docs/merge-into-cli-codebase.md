I want to merge this standalone cli into the standard Databricks CLI, which is located under /Users/fabian.jakobs/Workspaces. Requirements:

- The ClI should be exposed as `databricks apps mcp` sub command
- Make all changes in the Databricks CLI repo in a new branch called `apps-mcp` based on the latest main branch
- I want to maintain revision history to replay all commits of the current repo into the Databricks CLI repo
- Integrate with the Databricks CLI infrastructure such as go build system, testing, argument parsing, logging, etc.

Create a high level plan for these changes and break it down into phases. Create markdown files under `./plans` for each phase.

Then create a master task using the `bd` CLI and for each phase create a sub task. Do not make any code changes