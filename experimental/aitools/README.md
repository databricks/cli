# Experimental AI Tools

`databricks experimental aitools` is the remaining experimental surface for coding-agent workflows.

Current commands:

- `databricks experimental aitools skills list`
- `databricks experimental aitools skills install [skill-name]`
- `databricks experimental aitools install [skill-name]`
- `databricks experimental aitools tools query`
- `databricks experimental aitools tools discover-schema`
- `databricks experimental aitools tools get-default-warehouse`

Current behavior:

- `skills install` installs Databricks skills for detected coding agents.
- `install` is a compatibility alias for `skills install`.
- `tools` exposes a small set of AI-oriented workspace helpers.

Removed behavior:

- there is no MCP server under `experimental aitools`
- the old `deploy` and `validate` flows were removed
- command names and behavior in this area are still experimental and may change
