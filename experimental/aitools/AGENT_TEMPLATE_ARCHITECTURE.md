# Agent Template Architecture: Design Decisions

## Overview

This document explains the architectural decisions around the agent template structure and its relationship to other app templates in the experimental AI tools.

## Current Architecture

### Template Structure

The CLI provides several template types under `experimental aitools tools init-template`:

- **app** - UI-focused web applications using AppKit (TypeScript/React/tRPC)
- **agent** - Conversational AI agents using OpenAI Agents SDK (Python)
- **job** - Python notebooks with wheel packages
- **pipeline** - Lakeflow Declarative Pipelines (SQL or Python)
- **empty** - Minimal bundle for custom resources

### Template Locations

- **Experimental templates** (app, agent): `experimental/aitools/templates/`
  - Fetched from GitHub by default
  - Support local override via environment variables
  - Active development, evolving rapidly

- **Stable templates** (job, pipeline, empty): `libs/template/templates/`
  - Bundled with CLI binary
  - Production-ready, stable APIs
  - Standard template system

## Should Agent Be a Sub-template of App?

### Context

Both "app" and "agent" templates ultimately deploy as Databricks Apps, but serve different purposes:

- **App (appkit)**: TypeScript/React frontend, tRPC API layer, SQL queries, UI-focused
- **Agent**: Python backend, OpenAI Agents SDK, MLflow tracing, conversational AI (no UI by default)

### Option 1: Keep Separate (Current Implementation) ✅

**Pros:**
- **Clear mental model**: "app" = UI apps, "agent" = AI agents
- **Independent evolution**: Can iterate on agent features without affecting UI apps
- **Simpler schemas**: Each template has focused, easy-to-understand configuration
- **No tech stack interference**: TypeScript/React vs Python/Agents SDK don't mix
- **Clearer documentation**: Each template has dedicated docs for its specific use case
- **Easier user experience**: Users know exactly what they're getting
- **Faster scaffolding**: No need to process complex conditionals

**Cons:**
- **Some duplication**: App deployment configuration repeated across templates
- **Multiple commands**: Two separate entry points for app-related templates
- **Inconsistent patterns**: Each template may evolve deployment patterns differently

### Option 2: Merge as Sub-templates

**Approach:** Use conditional logic like `default-python` template:

```json
{
  "properties": {
    "app_type": {
      "type": "string",
      "enum": ["ui", "agent", "streamlit", "gradio"],
      "description": "Type of app to create"
    }
  }
}
```

With `__preamble.tmpl`:
```go
{{$is_ui_app := eq .app_type "ui"}}
{{$is_agent := eq .app_type "agent"}}

{{if not $is_ui_app}}
  {{skip "{{.project_name}}/src/frontend"}}
{{end}}

{{if not $is_agent}}
  {{skip "{{.project_name}}/agent_server"}}
{{end}}
```

**Pros:**
- **Single entry point**: One "app" command for all app types
- **Shared deployment patterns**: Consistent app configuration across types
- **Extensible**: Easy to add new app types (Streamlit, Gradio, Flask)
- **Category clarity**: Everything under "app" is an app

**Cons:**
- **Complex schema**: Many conditional options confuse users
- **Divergent tech stacks**: TypeScript/React + Python/Agents in one template is messy
- **Maintenance burden**: Changes to one app type risk breaking others
- **"Mega-template" anti-pattern**: Single template trying to do too much
- **Slower scaffolding**: Must process many conditionals even for simple cases
- **Documentation complexity**: Docs must explain all app types and their differences
- **Testing complexity**: Must test all combinations of conditionals

## Recommendation: Keep Separate + Future Category System

### Current (Recommended)

Keep templates separate as they are now:

```bash
databricks experimental aitools tools init-template app --name my-ui-app
databricks experimental aitools tools init-template agent --name my-agent
```

**Why:**
- Templates have fundamentally different tech stacks and purposes
- Each template can evolve independently
- Clear, focused user experience
- Simpler maintenance and testing

### Future Enhancement: Category-Based Discovery

If we need better organization, introduce a category system for discovery without merging templates:

```bash
# Discovery
databricks experimental aitools tools init-template list --category apps
# Shows: app (ui), agent (ai), streamlit (dashboards), gradio (ml-demos)

# Scaffolding (same as today)
databricks experimental aitools tools init-template app --name my-ui-app
databricks experimental aitools tools init-template agent --name my-agent
databricks experimental aitools tools init-template streamlit --name my-dashboard
```

This provides:
- ✅ Discoverability via categories
- ✅ Consistent app deployment experience
- ✅ Independent template maintenance
- ✅ Clear user experience
- ✅ Extensibility for new app types

## How Standard Templates Handle Multiple Use Cases

The `default-python` template provides a reference implementation:

### Schema-Based Feature Flags

```json
{
  "include_job": {"enum": ["yes", "no"]},
  "include_pipeline": {"enum": ["yes", "no"]},
  "include_python": {"enum": ["yes", "no"]},
  "language": {"enum": ["python", "sql"]}
}
```

### Conditional File Inclusion

```go
{{$pipeline := eq .include_pipeline "yes"}}
{{$notebook_job := eq .include_job "yes"}}

{{if not $notebook_job}}
  {{skip "{{.project_name}}/src/sample_notebook.ipynb"}}
{{end}}

{{if not $pipeline}}
  {{skip "{{.project_name}}/resources/{{.project_name}}_etl.pipeline.yml"}}
{{end}}
```

### When to Use This Pattern

✅ **Good for:**
- Templates with similar tech stacks (all Python, or all TypeScript)
- Variations that share most files (80%+ overlap)
- Features that can be mixed and matched
- Projects where users commonly want multiple features together

❌ **Bad for:**
- Templates with different tech stacks (Python + TypeScript)
- Variations that share few files (<50% overlap)
- Mutually exclusive project types
- Complex conditional logic that confuses users

## Decision Matrix

| Factor | Keep Separate | Merge with Conditionals |
|--------|--------------|------------------------|
| Tech stack similarity | Different (TS vs Python) ✅ | Same/similar stack ✅ |
| File overlap | <30% | >80% ✅ |
| User mental model | Clear ✅ | Complex |
| Maintenance complexity | Low ✅ | High |
| Testing complexity | Low ✅ | High |
| Extensibility | Medium | High ✅ |
| Documentation clarity | High ✅ | Medium |

**Conclusion:** For agent + app templates, "Keep Separate" is the right choice based on:
- Different tech stacks (TypeScript/React vs Python)
- Minimal file overlap (~20%)
- Clear mental models (UI app vs AI agent)

## Implementation Notes

### Environment Variable Overrides

Both experimental templates support local development overrides:

```bash
# App template
export DATABRICKS_APPKIT_TEMPLATE_PATH="/path/to/local/appkit"
databricks experimental aitools tools init-template app --name my-app

# Agent template
export DATABRICKS_AGENT_TEMPLATE_PATH="/path/to/local/agent"
databricks experimental aitools tools init-template agent --name my-agent
```

This pattern is **only for experimental templates** - stable templates (job, pipeline, empty) are bundled with the CLI and don't need it.

### Template Constants

Each template command defines:

```go
const (
    templateRepo = "https://github.com/databricks/cli"
    templateDir  = "experimental/aitools/templates/agent-openai-agents-sdk"
    branch       = "main"
    pathEnvVar   = "DATABRICKS_AGENT_TEMPLATE_PATH"
)
```

This allows:
- Default: Fetch from GitHub main branch
- Override: Use local path via environment variable

## Future Considerations

### Adding New App Types

When adding new app types (Streamlit, Gradio, Flask):

1. **Evaluate tech stack**: Similar to existing? → Consider sub-template. Different? → New template.
2. **Check file overlap**: >80%? → Consider sub-template. <50%? → New template.
3. **Assess user needs**: Do users commonly want both together? → Sub-template. Separate use cases? → New template.

### Example: Streamlit Dashboard Template

**Tech stack:** Python (similar to agent, different from app)
**File overlap:** ~30% with agent (databricks.yml, app.yaml, deployment)
**User needs:** Usually separate from conversational agents

**Decision:** Create separate `streamlit` template

```bash
databricks experimental aitools tools init-template streamlit --name my-dashboard
```

### Example: Agent with UI

**Tech stack:** Python + TypeScript (hybrid)
**File overlap:** Needs both agent and app files
**User needs:** Common use case - deploy agent with chat UI

**Decision:** This is a good candidate for a new combined template

```bash
databricks experimental aitools tools init-template agent-ui --name my-agent-app
```

This would:
- Include agent backend (Python/OpenAI SDK)
- Include chat frontend (TypeScript/React)
- Use existing agent + app templates as reference
- Be a new template, not a sub-template of either

## References

- Standard template system: `libs/template/`
- Experimental templates: `experimental/aitools/templates/`
- Template schema reference: `databricks_template_schema.json` in each template
- Conditional logic examples: `libs/template/templates/default/template/__preamble.tmpl`
