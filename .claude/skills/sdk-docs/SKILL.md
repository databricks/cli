---
name: databricks-sdk-docs
description: Use this skill when the user asks about Databricks SDK methods, API signatures, parameter types, return types, or how to use specific Databricks APIs programmatically. Triggers on questions like "how do I create a job", "what parameters does X take", "SDK method for Y", or "JobSettings fields".
allowed-tools: mcp__databricks-mcp__databricks_query_sdk_docs
---

# Databricks SDK Documentation Skill

When users ask about Databricks SDK usage, API methods, or type definitions, use the `databricks_query_sdk_docs` MCP tool to find accurate documentation.

## When to Use This Skill

- User asks "how do I create a job/cluster/pipeline using the SDK?"
- User needs method signatures: "what's the signature for Jobs.Create?"
- User asks about type fields: "what fields does CreateJob have?"
- User needs enum values: "what are the possible run lifecycle states?"
- User is confused about SDK API parameters or return types

## How to Query

Use the `databricks_query_sdk_docs` tool with these parameters:

```json
{
  "query": "search terms",
  "category": "methods|types|enums|services",  // optional filter
  "service": "jobs|clusters|pipelines|...",    // optional filter
  "limit": 10                                   // default 10, max 50
}
```

## Example Queries

| User Question | Tool Query |
|---------------|------------|
| "How do I create a job?" | `{"query": "create job", "category": "methods"}` |
| "What fields does JobSettings have?" | `{"query": "JobSettings", "category": "types"}` |
| "What are the run states?" | `{"query": "run lifecycle state", "category": "enums"}` |
| "List all jobs API methods" | `{"query": "jobs", "service": "jobs", "category": "methods"}` |

## Response Guidelines

After querying, provide:
1. The method signature with parameter types
2. A brief description of what the method does
3. Key parameters the user likely needs
4. A simple code example if applicable

Keep responses focused on what the user asked - don't dump all documentation.

## CLI Fallback

If MCP is unavailable, use the helper script:

```bash
# From the CLI repo root
.claude/skills/sdk-docs/query-sdk-docs.sh "create job"
.claude/skills/sdk-docs/query-sdk-docs.sh "JobSettings" types
.claude/skills/sdk-docs/query-sdk-docs.sh "list" methods jobs 20
```

The script searches the embedded SDK docs index directly using `jq`.
