# Context Management for Databricks MCP

## Current vs Proposed

| Aspect | Current | Proposed |
|--------|---------|----------|
| **First-call injection** | Middleware prepends init message to any first tool call — position-dependent, can't recover if lost | Explicit `databricks_discover` tool — agent controls when to call, can re-call anytime |
| **Context layers** | Mixed: `apps.tmpl` contains both target-specific (validation) and template-specific (TypeScript SDK) guidance | Separated: L1 (flow), L2 (target), L3 (template) with clear boundaries |
| **Existing projects** | No detection — treats all projects the same | Detectors identify target types from `databricks.yml`, inject relevant L2 |
| **Combined bundles** | Not supported | Injects all relevant L2 layers (apps + jobs) |
| **Template guidance** | Agent must be told to "read CLAUDE.md" via instruction | `init-template` returns CLAUDE.md content directly |
| **Extensibility** | Hardcoded capability check for "apps" | `DetectorRegistry` — add new detectors without changing core logic |

## Problem

Current implementation has several issues with how context is provided to AI agents:

1. **Mixed context layers** — target-specific guidance (apps) is mixed with template-specific guidance (TypeScript SDK)
2. **First-call injection is fragile** — middleware prepends initialization to whatever tool runs first, which is position-dependent and non-recoverable
3. **No project detection** — system doesn't adapt to existing projects or combined bundles (app + job)
4. **Template guidance is external** — agent must be instructed to read CLAUDE.md separately

## Design Goals

- Universal MCP for any coding agent (Claude, Cursor, etc.)
- Support multiple target types: apps, jobs, pipelines
- Support multiple templates per target type
- Clean separation of context layers
- Detect existing project context automatically

## Context Layers

| Layer | Content | When Injected |
|-------|---------|---------------|
| **L1: Flow** | Universal workflow, available tools, CLI patterns | Always (via `databricks_discover`) |
| **L2: Target** | Target-specific: validation, deployment, constraints | When target type detected (existing project) or after `init-template` (new project) |
| **L3: Template** | SDK/language-specific: file structure, commands, patterns | After `init-template` only. For existing projects, agent reads CLAUDE.md naturally. |

### Examples

**L1 (universal):** "validate before deploying", "use invoke_databricks_cli for all commands"

**L2 (apps):** app naming constraints, deployment consent requirement, app-specific validation

**L3 (appkit-typescript):** npm scripts, tRPC patterns, useAnalyticsQuery usage, TypeScript import rules

## Architecture

### Tools

| Tool | Purpose |
|------|---------|
| `databricks_discover` | Returns L1 + detects project to inject L2/L3. **Agents call this first.** |
| `databricks_configure_auth` | Switch workspace profile/host |
| `invoke_databricks_cli` | Execute CLI commands |

The `databricks_discover` tool description explicitly instructs agents to call it first during planning. This replaces the first-call middleware injection.

### Detector Registry

Extensible system for detecting project context:

```
DetectorRegistry
├── BundleDetector      → InProject, TargetTypes (from databricks.yml)
├── TemplateDetector    → Template (from package.json, pyproject.toml, etc.)
└── ... future detectors
```

Detectors run in sequence, each contributing to `DetectedContext`:

```go
type DetectedContext struct {
    InProject   bool
    TargetTypes []string          // ["apps", "jobs"] - supports combined bundles
    Template    string            // "appkit-typescript", "python", etc.
    BundleInfo  *BundleInfo
    Metadata    map[string]string // extensible
}
```

Note: `TemplateDetector` identifies the template type but does NOT read CLAUDE.md. For existing projects, agents can read project files naturally. L3 is only injected by `init-template` when scaffolding new projects.

### Template Structure

```
lib/prompts/
├── flow.tmpl              # L1
├── target_apps.tmpl       # L2
├── target_jobs.tmpl       # L2
└── target_pipelines.tmpl  # L2

templates/appkit/template/{{.project_name}}/
└── CLAUDE.md              # L3 (injected after scaffold)
```

## Flow

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
  │   ["experimental", "apps-mcp", "tools", "init-template", ...]
  │                              ├─► Scaffold project
  │                              ├─► Determine target type (apps)
  │                              ├─► Read generated CLAUDE.md
  │◄─────────────────────────────┼─► Return success + L2[apps] + L3
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
  │                              ├─► TemplateDetector: found appkit-typescript
  │                              ├─► Assemble L1 + L2[apps] + L2[jobs]
  │◄─────────────────────────────┤
  │                              │
  │   (agent has L1 + L2)        │
  │                              │
  ├─► Read CLAUDE.md, explore    │
  │   project files naturally    │
  │                              │
  │   (agent learns L3 itself)   │
```

### Combined Bundles

When `databricks.yml` contains multiple resource types (e.g., app that requires a daily job), all relevant L2 templates are injected:

```go
for _, target := range detected.TargetTypes {
    result += prompts.MustExecuteTemplate(fmt.Sprintf("target_%s.tmpl", target), data)
}
```

## Migration

1. Remove `EngineGuideMiddleware`
2. Rename `explore` → `databricks_discover`, add `working_directory` param
3. Split current `apps.tmpl` into `target_apps.tmpl` (L2) and keep SDK details in CLAUDE.md (L3)
4. Create `flow.tmpl` from current `initialization_message.tmpl` content
5. Modify `init-template` to return L2[target] + L3 (CLAUDE.md content) in response
6. Implement `DetectorRegistry` with `BundleDetector` and `TemplateDetector`

## Notes

- Agents can re-request L1+L2 anytime by calling `databricks_discover` again
- For L3, agents read CLAUDE.md directly (no special mechanism needed)
