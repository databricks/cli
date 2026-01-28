# Agent Template Discovery Fix

## Issue Summary

When Claude Code users asked to create an agent, Claude would incorrectly use the `app` template instead of the `agent` template, resulting in errors like:

```
Error: validation for template input parameters failed.
no value provided for required property sql_warehouse_id
```

## Root Cause

The `databricks_discover` MCP tool output did not include the `agent` template in its scaffolding examples. The output showed:

```
For apps:
invoke_databricks_cli 'experimental aitools tools init-template app --name my-app'

For jobs (Python notebooks with wheel package):
invoke_databricks_cli 'experimental aitools tools init-template job --name my_job'

For pipelines (Lakeflow Declarative Pipelines):
invoke_databricks_cli 'experimental aitools tools init-template pipeline --name my_pipeline'

For custom resources (dashboards, alerts, etc.):
invoke_databricks_cli 'experimental aitools tools init-template empty --name my_project'
```

**Missing:** The agent template example!

When Claude saw a request like "create an agent" or "scaffold an agent project", it would:
1. Look at the databricks_discover examples
2. Not find an `agent` template example
3. Default to using the `app` template (closest match)
4. Hit the SQL warehouse validation error since apps require a warehouse but the user didn't provide one

## Why SQL Warehouse Was Required for Apps

The `app` template (appkit) requires a SQL warehouse because:

1. **Template schema defines it**: `experimental/aitools/templates/appkit/databricks_template_schema.json`
   ```json
   {
     "properties": {
       "sql_warehouse_id": {
         "type": "string",
         "description": "SQL Warehouse ID",
         "order": 2
       }
     }
   }
   ```

2. **Validation enforces all properties**: `libs/template/config.go`
   ```go
   func (c *config) validate() error {
       // For final validation, all properties in the JSON schema
       // should have a value defined.
       c.schema.Required = maps.Keys(c.schema.Properties)
       if err := c.schema.ValidateInstance(c.values); err != nil {
           return fmt.Errorf("validation for template input parameters failed. %w", err)
       }
       return nil
   }
   ```

3. **AppKit needs SQL access**: The appkit template creates TypeScript/React apps that query SQL warehouses via tRPC, so a warehouse is a core requirement.

## Why Agent Template Doesn't Need SQL Warehouse

The `agent` template (OpenAI Agents SDK) has different requirements:

1. **Template schema is minimal**: `experimental/aitools/templates/agent-openai-agents-sdk/databricks_template_schema.json`
   ```json
   {
     "properties": {
       "project_name": {
         "type": "string",
         "description": "Project Name",
         "order": 1
       },
       "mlflow_experiment_name": {
         "type": "string",
         "description": "MLflow Experiment Name (Optional)",
         "default": "",
         "order": 2
       }
     }
   }
   ```

2. **Agents use MCP, not SQL directly**: Agents access data through MCP servers (Unity Catalog functions, code interpreter), not by directly querying SQL warehouses.

3. **Different architecture**: Agents are Python-based conversational AI, while apps are TypeScript/React data dashboards.

## Solution

Added the agent template to the `databricks_discover` output in `experimental/aitools/lib/prompts/flow.tmpl`:

```diff
 For apps:
 invoke_databricks_cli 'experimental aitools tools init-template app --name my-app --description "My app description"'

+For agents (Conversational AI with OpenAI Agents SDK):
+invoke_databricks_cli 'experimental aitools tools init-template agent --name my_agent'
+Note: Creates Python agent with OpenAI Agents SDK, Claude model, MLflow tracing, and MCP integration
+
 For jobs (Python notebooks with wheel package):
 invoke_databricks_cli 'experimental aitools tools init-template job --name my_job'
```

Also updated the naming conventions note:
```diff
 Notes:
 - App name must be ≤26 characters (dev- prefix adds 4 chars, max total 30)
-- Job/pipeline/project names: letters, numbers, underscores only
+- Agent/job/pipeline/project names: letters, numbers, underscores only
```

## Impact

After this fix:
- ✅ Claude Code sees the agent template option in databricks_discover
- ✅ When users ask to "create an agent", Claude uses the correct template
- ✅ No SQL warehouse validation errors for agent projects
- ✅ Clear distinction between app (UI + SQL) and agent (AI + MCP) templates

## Testing

To verify the fix works:

1. **Restart Claude Code** to pick up the new CLI build
2. **Ask Claude**: "Create a new agent called test_agent"
3. **Expected behavior**: Claude uses `experimental aitools tools init-template agent --name test_agent`
4. **Expected output**: Agent project scaffolded without SQL warehouse errors

### Before Fix

```
User: Create a new agent
Claude: I'll create an app for you
→ invoke_databricks_cli 'experimental aitools tools init-template app --name test-agent'
→ Error: no value provided for required property sql_warehouse_id
```

### After Fix

```
User: Create a new agent
Claude: I'll create an agent for you
→ invoke_databricks_cli 'experimental aitools tools init-template agent --name test_agent'
→ ✅ Success: Your new agent project has been created
```

## Related Files

- **Discovery template**: `experimental/aitools/lib/prompts/flow.tmpl` (updated)
- **Agent template schema**: `experimental/aitools/templates/agent-openai-agents-sdk/databricks_template_schema.json`
- **App template schema**: `experimental/aitools/templates/appkit/databricks_template_schema.json`
- **Discovery implementation**: `experimental/aitools/lib/providers/clitools/discover.go`
- **Template validation**: `libs/template/config.go`

## Key Learnings

1. **Discovery is critical**: The `databricks_discover` output is the primary way Claude Code learns about available templates. Missing examples = missing functionality in Claude's eyes.

2. **Template requirements vary**: Different templates have different required properties. Apps need SQL warehouses, agents don't. Make sure examples reflect these differences.

3. **Validation is strict**: The template system marks ALL schema properties as required during validation. Properties without defaults must be explicitly provided.

4. **Template vs MCP requirements**: Template scaffolding requirements (from schema) are separate from MCP server startup requirements (--warehouse-id flag). The MCP server can have a warehouse for SQL queries without requiring all templates to use it.

## Future Improvements

Consider adding more context to databricks_discover about when to use each template:

```
For apps (UI-focused data dashboards):
- Use when: Building interactive dashboards with SQL queries
- Tech stack: TypeScript/React/tRPC
- Requires: SQL warehouse for data access
invoke_databricks_cli 'experimental aitools tools init-template app --name my-app'

For agents (Conversational AI):
- Use when: Building AI assistants, chatbots, or agentic workflows
- Tech stack: Python/OpenAI Agents SDK/MLflow
- Requires: No SQL warehouse (uses MCP for data access)
invoke_databricks_cli 'experimental aitools tools init-template agent --name my_agent'
```

This would help Claude Code make better decisions about which template to use based on user intent.
