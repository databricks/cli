---
description: Comment conventions for source files
globs:
  - "**/*.go"
  - "**/*.py"
  - "**/*.pyi"
  - "**/*.sh"
  - "**/*.scala"
  - "**/*.js"
  - "**/*.java"
  - "**/*.sql"
  - "**/*.r"
  - "**/*.yml"
  - "**/*.yaml"
paths:
  - "**/*.go"
  - "**/*.py"
  - "**/*.pyi"
  - "**/*.sh"
  - "**/*.scala"
  - "**/*.js"
  - "**/*.java"
  - "**/*.sql"
  - "**/*.r"
  - "**/*.yml"
  - "**/*.yaml"
---

# Comment conventions

**RULE: Do not modify or remove existing comments in code you didn't write**, unless the user asks or your change makes them wrong. They often encode non-obvious context (a bug reference, a workaround).

**RULE: Comments explain "why", not "what". Keep them concise.** Say only what the code cannot; a doc comment gives the caller the contract and any surprising behavior, not a line-by-line re-spec.

**RULE: When code relies on a non-obvious invariant, workaround, or backend quirk, add a short comment stating the reason.** Missing comments are the most common thing reviewers catch: API quirks (PATCH-like semantics, no get-by-name, stripped prefixes), fields intentionally included or excluded (output-only, etag, `ForceSendFields`), branches that look dead but are kept as guards, and test expectations that aren't obvious from the assertions.

```go
// The Workspace API strips the "/Workspace" prefix from parent_path on GET,
// so we re-add it here to match the local configuration.
parentPath = "/Workspace" + parentPath
```

**RULE: When integrating external tools or detecting environment variables, include source reference URLs as comments.**
