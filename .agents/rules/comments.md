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

**RULE: Do not modify or remove existing comments in code you didn't write.** Comments often encode non-obvious context (a bug reference, a workaround, a reason the code is shaped a certain way) that is lost if rewritten. Leave them alone unless the user explicitly asks for a change.

**RULE: Comments should explain "why", not "what".** Reviewers consistently reject comments that merely restate the code.

**RULE: Keep comments concise, and avoid redundant or verbose ones.** Say only what the code cannot, then stop — only add a comment if it complements the code rather than repeating it or its signature. A doc comment gives the caller the contract and any surprising behavior, not a line-by-line re-spec. AI-generated comments trend long and explanatory, so tighten them before committing.

**RULE: When code relies on a non-obvious invariant, workaround, or backend quirk, add a short comment stating the reason.** The inverse of the rule above: noise comments are bad, but missing comments are the single most common thing reviewers catch. Triggers include: API quirks (PATCH-like semantics, no get-by-name, stripped prefixes), fields intentionally included or excluded (output-only, etag, `ForceSendFields`), branches that look dead but are kept as guards, and tests where the expectation isn't obvious from the assertions.

GOOD:

```go
// The Workspace API strips the "/Workspace" prefix from parent_path on GET,
// so we re-add it here to match the local configuration.
parentPath = "/Workspace" + parentPath
```

BAD:

```go
parentPath = "/Workspace" + parentPath
```

**RULE: In Go, document functions with a doc comment that starts with the function name and ends with a period.**

```go
// SomeFunc does something.
func SomeFunc() {
	...
}
```

**RULE: When integrating external tools or detecting environment variables, include source reference URLs as comments.** This lets future readers trace where the behavior came from.
