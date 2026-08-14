# Code conventions

Conventions that apply across the whole repository, regardless of language. Read this before you write or change code.

Rules prefixed `**RULE:**` are mandatory. `GOOD:` and `BAD:` labels on code snippets mark patterns to follow and patterns to avoid.

## Comments

**RULE: Do not modify or remove existing comments in code you didn't write.** Comments often encode non-obvious context (a bug reference, a workaround, a reason the code is shaped a certain way) that is lost if rewritten. Leave them alone unless the user explicitly asks for a change.

**RULE: Comments should explain "why", not "what".** Reviewers consistently reject comments that merely restate the code.

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

## Language- and domain-specific rules

The rules above apply everywhere. Before touching files in these areas, also read the matching rule file under `.agent/rules/`. These are the source of the per-file rules that Cursor auto-attaches by glob; other agents (Claude Code included) don't auto-load them, so read them directly when your change falls in scope.

| When you work on… | Read |
| --- | --- |
| Go code (`**/*.go`) | [.agent/rules/style-guide-go.md](.agent/rules/style-guide-go.md) |
| Python code (`**/*.py`) | [.agent/rules/style-guide-py.md](.agent/rules/style-guide-py.md) |
| Tests (`**/*_test.go`, `acceptance/**`, `integration/**`) | [.agent/rules/testing.md](.agent/rules/testing.md) |
| Deploy telemetry (`libs/telemetry/**`, `bundle/metrics/**`, `bundle/phases/telemetry.go`) | [.agent/rules/telemetry.md](.agent/rules/telemetry.md) |
| Direct-engine resources (`bundle/direct/dresources/**/*.go`) | [.agent/rules/dresources.md](.agent/rules/dresources.md) |
| Changelog entries (`CHANGELOG.md`, `.nextchanges/**`) | [.agent/rules/changelog.md](.agent/rules/changelog.md) |
| Auto-generated files (`cmd/workspace/**`, `cmd/account/**`, `internal/mocks/**`, generated schemas) | [.agent/rules/auto-generated-files.md](.agent/rules/auto-generated-files.md) |
| Template schema (`**/databricks_template_schema.json`) | [.agent/rules/databricks-template-schema-json.md](.agent/rules/databricks-template-schema-json.md) |
