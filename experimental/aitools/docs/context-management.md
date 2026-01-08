<!-- DO NOT MODIFY: This documentation defines the context management architecture for Databricks aitools -->
# Context Management for Databricks aitools

## Goals

- Universal MCP for any coding agent (Claude, Cursor, etc.)
- Support multiple target types: apps, jobs, pipelines
- Support multiple templates per target type
- Clean separation of context layers
- Detect existing project context automatically

## Context Layers

| Layer | Content | When Injected |
|-------|---------|---------------|
| **L0: Tools** | Databricks aitools tool names and descriptions | Always (MCP protocol) |
| **L1: Flow** | Universal workflow, available tools, CLI patterns | Always (via `databricks_discover`) |
| **L2: Target** | Target-specific: validation, deployment, constraints | When target type detected or after `init-template` |
| **L3: Template** | SDK/language-specific: file structure, commands, patterns | After `init-template`. For existing projects, agent reads CLAUDE.md. |

L0 is implicit - tool descriptions guide agent behavior before any tool is called (e.g., `databricks_discover` description tells agent to call it first during planning).

### Examples

**L1 (universal):** "validate before deploying", "use invoke_databricks_cli for all commands"

**L2 (apps):** app naming constraints, deployment consent requirement, app-specific validation

**L3 (appkit-typescript):** npm scripts, tRPC patterns, useAnalyticsQuery usage, TypeScript import rules

## Flows

### New Project

```
Agent                           MCP
  │                              │
  ├─► databricks_discover        │
  │   {working_directory: "."}   │
  │                              ├─► Run detectors (nothing found)
  │                              ├─► Return L1 only
  │◄─────────────────────────────┤
  │                              │
  ├─► invoke_databricks_cli      │
  │   ["...", "init-template", ...]
  │                              ├─► Scaffold project
  │                              ├─► Return L2[apps] + L3
  │◄─────────────────────────────┤
  │                              │
  ├─► (agent now has L1 + L2 + L3)
```

### Existing Project

```
Agent                           MCP
  │                              │
  ├─► databricks_discover        │
  │   {working_directory: "./my-app"}
  │                              ├─► BundleDetector: found apps + jobs
  │                              ├─► Return L1 + L2[apps] + L2[jobs]
  │◄─────────────────────────────┤
  │                              │
  ├─► Read CLAUDE.md naturally   │
  │   (agent learns L3 itself)   │
```

### Combined Bundles

When `databricks.yml` contains multiple resource types (e.g., app + job), all relevant L2 layers are injected together.

## Extensibility

New target types can be added by:
1. Creating `target_<type>.tmpl` in `lib/prompts/`
2. Adding detection logic to recognize the target type from `databricks.yml`

New templates can be added by:
1. Creating template directory with CLAUDE.md (L3 guidance)
2. Adding detection logic to recognize the template from project files
