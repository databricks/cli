---
name: pr-checklist
description: Checklist to run before submitting a PR
---

Before submitting a PR, run these commands to match what CI checks. CI uses the **full** variants (not the diff-only wrappers), so `make lint` alone is insufficient.

```bash
# 1. Formatting and checks (CI runs fmtfull, not fmt)
make fmtfull
make checks

# 2. Linting (CI runs full golangci-lint, not the diff-only wrapper)
make lintfull

# 3. Tests (CI runs with both deployment engines)
make test

# 4. If you changed bundle config structs or schema-related code:
make schema

# 5. If you changed files in python/:
cd python && make codegen && make test && make lint && make docs

# 6. If you changed experimental/aitools or experimental/ssh:
make test-exp-aitools   # only if aitools code changed
make test-exp-ssh       # only if ssh code changed
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
