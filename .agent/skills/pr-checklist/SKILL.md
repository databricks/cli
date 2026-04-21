---
name: pr-checklist
description: Checklist to run before submitting a PR
---

Before submitting a PR, run these commands to match what CI checks. CI uses the full variants (not the `-q` diff-only wrappers), so `./task lint-q` alone is insufficient.

```bash
# 1. Formatting and checks (CI runs fmt, not fmt-q)
./task fmt
./task checks

# 2. Linting (CI runs full golangci-lint across all modules, not the diff-only wrapper)
./task lint

# 3. Tests (CI runs with both deployment engines)
./task test

# 4. If you changed bundle config structs or schema-related code:
./task generate-schema

# 5. If you changed files in python/:
./task pydabs-codegen pydabs-test pydabs-lint pydabs-docs

# 6. If you changed experimental/aitools or experimental/ssh:
./task test-exp-aitools   # only if aitools code changed
./task test-exp-ssh       # only if ssh code changed
```

## Final cleanup scan

After the commands above pass, scrub the diff before pushing. The quick version: run `git diff @{u}` and read through what you added. Specifically:

- **Debug prints**: look for newly added `fmt.Print`, `fmt.Printf`, `fmt.Println`, `log.Print`, `log.Printf`, `log.Println`, or bare `println(...)` calls. A regex that scans only added lines against your upstream branch:

  ```bash
  git diff @{u} -- '*.go' | rg '^\+.*\b(fmt|log)\.(Print|Printf|Println)\b|^\+.*\bprintln\('
  ```

  If you have no upstream yet, substitute the intended base (e.g. `origin/main`) for `@{u}`.
- **Commented-out code**: delete it. If it's needed for reference, it lives in git history.
- **TODOs without a ticket**: either add a ticket reference (e.g. `// TODO(DECO-1234): ...`) or remove the TODO. Un-tracked TODOs rot.
- **Unintended files**: review `git status` and `git diff --stat` to confirm only the files you meant to change are staged.

## PR description

Follow `.github/PULL_REQUEST_TEMPLATE.md` exactly. Use its section headings (`## Changes`, `## Why`, `## Tests`) in the same order, and fill each one in. Do not invent new sections (`## Summary`, `## Test plan`, etc.), do not drop sections, and do not leave the HTML comment placeholders in the final body — replace them with real content. If a section genuinely does not apply (e.g. a docs-only change has no test steps), say so explicitly under that heading rather than removing it.

When using `gh pr create`, read `.github/PULL_REQUEST_TEMPLATE.md` first and base `--body` on it.

If an agent (you) authored or substantially helped author the PR, disclose it on the last line of the body, e.g. `_This PR was written by Claude Code._` or `_PR description drafted with Claude Code._`. Be honest about the level of involvement — "written by" vs. "drafted with" vs. "reviewed by" — and keep it to a single italicized line so it doesn't crowd the template sections.

## Changelog entry

Add a `NEXT_CHANGELOG.md` entry when your change is user-visible. CI generates the real `CHANGELOG.md` from `NEXT_CHANGELOG.md` at release time, so never hand-edit `CHANGELOG.md` directly.

**When to add an entry:**
- New or changed CLI command, flag, or subcommand behavior
- New or changed bundle config field, schema, or engine behavior
- New direct dependency (annotate under `Dependency updates`)
- Bug fix that users will notice

**When to skip:**
- Experimental commands (under `experimental/`): no entry until the feature graduates out of experimental
- Pure refactors, internal renames, test-only changes, and doc-only changes
- Auto-generated output changes without a corresponding user-facing change

**How to add:**
- Pick the right section (`CLI`, `Bundles`, `Dependency updates`) under the current `## Release vX.Y.Z` header.
- One or two sentences, user-facing language, no Jira links.
- Reference the PR number once it's open: after `gh pr create`, edit the entry to append ` (#NNNN)` or similar matching nearby entries.
- Match the voice and tense of the existing entries in the file.
