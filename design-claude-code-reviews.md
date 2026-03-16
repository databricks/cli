# Design: Claude Code for Databricks CLI

## Objective

AI-assisted PR reviews and interactive `@claude` mentions on `databricks/cli` and `databricks-eng/eng-dev-ecosystem` using Claude Code CLI backed by Claude Opus 4.6 on Databricks Model Serving. Authentication via GitHub OIDC federation — no static API keys.

## Features

1. **Auto-review on PR open**: Claude reviews every PR opened by an org member, posting inline review comments.
2. **`@claude` mentions**: Maintainers comment `@claude <instruction>` to request code changes, answers, or commands. Claude pushes commits directly.
3. **Supported repos**: `databricks/cli` (cross-repo dispatch) and `databricks-eng/eng-dev-ecosystem` (direct).

## Architecture

### CLI repo (cross-repo dispatch)

```
databricks/cli                              databricks-eng/eng-dev-ecosystem
================                            ================================

PR opened / @claude comment
  -> claude-code.yml (thin dispatcher)
     runs-on: databricks-deco-testing-runner-group
     -> GitHub App token (DECO_WORKFLOW_TRIGGER)
     -> gh workflow run cli-claude-code.yml
     -> Track remote run (gh run watch)
                                            cli-claude-code.yml (workflow_dispatch)
                                              runs-on: databricks-eng-protected-runner-group
                                              environment: integration-tests
                                              -> Checkout eng-dev-ecosystem (for composite action)
                                              -> Checkout databricks/cli PR branch to cli/
                                              -> Build prompt (review or assist)
                                              -> .github/actions/claude-code (composite action)
                                                 -> PR helper scripts
                                                 -> OIDC token refresh helper
                                                 -> Validate connectivity
                                                 -> Claude Code CLI
                                                    -> Model Serving (Claude Opus 4.6)
                                              -> Post review / push commits via DECO_GITHUB_TOKEN
```

CLI runs on GitHub-hosted runners not allowlisted in the Databricks account IP ACL. The OIDC token exchange endpoint blocks non-allowlisted IPs, so CLI dispatches to `eng-dev-ecosystem` which uses self-hosted runners with allowlisted IPs.

The CLI dispatcher tracks the remote run via `gh run watch --exit-status`, keeping the PR check visible while Claude works and mirroring the exit status. A step summary links to the actual eng-dev-ecosystem run.

### eng-dev-ecosystem repo (direct)

```
databricks-eng/eng-dev-ecosystem
================================

PR opened / @claude comment
  -> claude-code-self.yml (direct trigger)
     runs-on: databricks-eng-protected-runner-group
     -> Checkout PR branch
     -> .github/actions/claude-code (composite action)
        -> PR helper scripts
        -> OIDC token refresh helper
        -> Validate connectivity
        -> Claude Code CLI
           -> Model Serving (Claude Opus 4.6)
     -> Post review / push commits via github.token
```

### Workflow and action files

| Repo | File | Purpose |
|---|---|---|
| `databricks/cli` | `.github/workflows/claude-code.yml` | Thin dispatcher — triggers eng-dev-ecosystem via GitHub App, tracks remote run |
| `databricks-eng/eng-dev-ecosystem` | `.github/actions/claude-code/action.yml` | Composite action — OIDC auth, token refresh, PR helpers, Claude Code CLI execution |
| `databricks-eng/eng-dev-ecosystem` | `.github/workflows/cli-claude-code.yml` | CLI-specific — builds prompts, checks out CLI code, calls composite action |
| `databricks-eng/eng-dev-ecosystem` | `.github/workflows/claude-code-self.yml` | Self-review on eng-dev-ecosystem's own PRs, calls composite action |

### Composite action (`.github/actions/claude-code/action.yml`)

Shared infrastructure used by both `cli-claude-code.yml` and `claude-code-self.yml`:

| Input | Required | Description |
|---|---|---|
| `prompt` | Yes | Instructions for Claude Code |
| `pr_number` | Yes | PR number for helper scripts |
| `repo` | No | Target repo for `--repo` flag (empty = current repo) |
| `gh_token` | Yes | GitHub token for PR operations |
| `claude_args` | No | Additional CLI arguments (e.g. `--max-turns 100`) |
| `working_directory` | No | Directory to run Claude in (default: `.`) |

Steps performed by the action:
1. Create PR-scoped helper scripts (`pr-comment`, `pr-push`, `pr-diff`, `pr-view`, `pr-review`)
2. Setup Node.js 20 and install `@anthropic-ai/claude-code`
3. Create OIDC token refresh helper at `/usr/local/bin/get-databricks-token`
4. Validate connectivity to Model Serving endpoint
5. Run Claude Code with hardcoded allowed tools and system prefix

The caller workflow must:
- Set `id-token: write` permission (for OIDC)
- Check out the target repository before calling the action
- Check out eng-dev-ecosystem as well (for cross-repo, so the action is available at `./.github/actions/claude-code`)

## Authentication

### OIDC token exchange

GitHub Actions requests an OIDC JWT (audience = Databricks account ID), exchanged for a Databricks OAuth token:

```
POST https://accounts.cloud.databricks.com/oidc/accounts/968367da-7edd-44f7-9dea-3e0b20b0ec97/v1/token
grant_type=urn:ietf:params:oauth:grant-type:token-exchange
subject_token=<github-oidc-jwt>
subject_token_type=urn:ietf:params:oauth:token-type:jwt
client_id=b76b6808-9e10-43b3-be20-6b6d19ed1af0
scope=all-apis
```

### Token refresh

GitHub OIDC tokens expire in ~5 minutes. Claude Code's `apiKeyHelper` mechanism calls a helper script every few minutes (or on 401) to re-exchange tokens, allowing sessions to run beyond the 5-minute token lifetime. The helper script is written to `/usr/local/bin/get-databricks-token` and configured via `~/.claude/settings.json`:

```json
{"apiKeyHelper": "/usr/local/bin/get-databricks-token"}
```

`ANTHROPIC_AUTH_TOKEN` is NOT set — it takes precedence over `apiKeyHelper` and would prevent refresh.

### OAuth scope

Uses `scope=all-apis`. The narrower `scope=serving.serving-endpoints` was tested but returns 403 from Model Serving — `all-apis` is required.

### Service principal

- **Name**: `DECO-TF-AWS-PROD-IS-SPN`
- **Service principal ID**: `3749120194272290`
- **Application/Client ID**: `b76b6808-9e10-43b3-be20-6b6d19ed1af0`

### Federation policies

Exact-match on the `job_workflow_ref` OIDC claim. Only workflows on `main` in eng-dev-ecosystem can obtain tokens.

| Policy ID | Matches | Purpose |
|---|---|---|
| `claude-code-github-oidc` | `...claude-code-self.yml@refs/heads/main` | Self-review (eng-dev-ecosystem PRs) |
| `claude-code-github-oidc-cli` | `...cli-claude-code.yml@refs/heads/main` | CLI PR review/assist |

Both policies use:
- **Issuer**: `https://token.actions.githubusercontent.com`
- **Audience**: `968367da-7edd-44f7-9dea-3e0b20b0ec97` (Databricks account ID)
- **Subject claim**: `job_workflow_ref`

### Cross-repo access

| Secret/Token | Location | Purpose |
|---|---|---|
| `DECO_WORKFLOW_TRIGGER` GitHub App | CLI `test-trigger-is` environment | Dispatch workflows to eng-dev-ecosystem and track runs |
| `DECO_GITHUB_TOKEN` PAT | eng-dev-ecosystem `integration-tests` environment | Write access to `databricks/cli` (push commits, post comments) |

## Model serving

- **Endpoint**: `https://dbc-1232e87d-9384.cloud.databricks.com/serving-endpoints/anthropic`
- **Model**: `databricks-claude-opus-4-6`
- **Auth**: `Authorization: Bearer` (via `apiKeyHelper`)
- **Environment variables**: `ANTHROPIC_BASE_URL`, `ANTHROPIC_MODEL`, `DISABLE_PROMPT_CACHING=1`, `CLAUDE_CODE_DISABLE_EXPERIMENTAL_BETAS=1`

## Security model

### Access control

Only GitHub org members/owners can trigger Claude. Uses `author_association` allowlists per [GitHub Security Lab guidance](https://securitylab.github.com/resources/github-actions-preventing-pwn-requests/):

```yaml
contains(fromJson('["MEMBER","OWNER"]'), github.event.pull_request.author_association)
```

### Trigger surface

**CLI repo (`databricks/cli`):**

| Trigger | Gate |
|---|---|
| `pull_request` (opened) | `MEMBER`/`OWNER` allowlist + fork guard (`!head.repo.fork`) |
| `issue_comment` (`@claude`) | `MEMBER`/`OWNER` allowlist + bot filter + PR-only guard + fork API check |
| `pull_request_review_comment` (`@claude`) | `MEMBER`/`OWNER` allowlist + bot filter |

**eng-dev-ecosystem (`databricks-eng/eng-dev-ecosystem`):**

| Trigger | Gate |
|---|---|
| `pull_request` (opened) | `MEMBER`/`OWNER` allowlist + fork guard |
| `issue_comment` (`@claude`) | `MEMBER`/`OWNER` allowlist + bot filter + PR-only guard |
| `pull_request_review_comment` (`@claude`) | `MEMBER`/`OWNER` allowlist + bot filter |

### Security measures

| Measure | Description |
|---|---|
| **Fork guard** | `!github.event.pull_request.head.repo.fork` on review; API check via `gh pr view --json isCrossRepository` on CLI assist |
| **Bot filter** | `github.event.comment.user.type != 'Bot'` prevents loops |
| **PR-only guard** | `github.event.issue.pull_request` ensures `issue_comment` only fires on PRs |
| **Concurrency groups** | Both `review` and `assist` jobs cancel in-progress runs on the same PR |
| **Hardcoded tool allowlist** | `--allowedTools` in the composite action restricts Claude to specific safe commands; callers cannot override |
| **Max turns** | `--max-turns 100` limits work per CLI invocation |
| **Timeout** | `timeout-minutes: 30` on all jobs caps session duration |
| **PR-scoped helpers** | `pr-comment`, `pr-push`, `pr-diff`, `pr-view`, `pr-review` hardcode PR number at creation time |
| **Branch name escaping** | `printf '%q'` in `pr-push` prevents shell injection via branch names |
| **Comment truncation** | Assist truncates `comment_body` to 4000 chars |
| **Heredoc delimiter randomization** | `PROMPT_$(openssl rand -hex 8)` prevents `comment_body` from breaking `GITHUB_OUTPUT` |
| **Script injection prevention** | `comment_body` and `pr_number` passed via `process.env` in `actions/github-script`, never via `${{ }}` in `run:` blocks |
| **Action SHA pinning** | Third-party actions (`create-github-app-token`, `github-script`) pinned to commit SHAs, not mutable tags |
| **Minimal dispatcher permissions** | CLI dispatcher only requests `contents: read`; write permissions are on the eng-dev-ecosystem job |
| **Environment-gated secrets** | `DECO_GITHUB_TOKEN` requires `integration-tests` environment approval; `DECO_WORKFLOW_TRIGGER` requires `test-trigger-is` |

### Allowed tools

The composite action hardcodes this tool allowlist for Claude Code:

```
Bash(make lint), Bash(make test), Bash(make fmt), Bash(make schema),
Bash(go build *), Bash(go test *), Bash(go vet), Bash(go vet *),
Bash(terraform fmt), Bash(terraform fmt *), Bash(terraform validate), Bash(terraform validate *),
Bash(git add *), Bash(git commit *), Bash(git diff), Bash(git diff *),
Bash(git log), Bash(git log *), Bash(git status), Bash(git show *),
Bash(pr-comment *), Bash(pr-diff), Bash(pr-diff *),
Bash(pr-push), Bash(pr-push *), Bash(pr-review *),
Bash(pr-view), Bash(pr-view *), Bash(grep *),
Read, Edit, Write, Glob, Grep
```

Notable exclusions: no `rm`, no `gh` (only via wrapper scripts), no arbitrary shell commands, no network tools.

### Threat model

| Threat | Mitigation |
|---|---|
| External user opens fork PR | `author_association` allowlist + fork guard |
| Attacker modifies `CLAUDE.md` in fork PR | Fork PR authors blocked by allowlist |
| External user comments `@claude` | `author_association` allowlist |
| Bot triggers Claude in loop | `user.type != 'Bot'` filter |
| Comment body shell injection | Passed via env vars, truncated, heredoc delimiter randomized |
| Comment body injected into workflow dispatch | `comment_body` passed via `process.env`, never `${{ }}` in shell |
| Claude targets wrong PR | Helper scripts hardcode PR number at creation time |
| Branch name injection | `printf '%q'` escapes branch name in `pr-push` |
| Internal user runs destructive commands | Hardcoded `--allowedTools` restricts available commands |
| Token leaked in logs | `::add-mask::` on tokens; token exchange errors only log HTTP status |
| Cost overrun | `--max-turns` + `timeout-minutes` |
| Mutable action tag supply chain attack | Actions pinned to commit SHAs (`create-github-app-token@fee1f7d...`, `github-script@f28e40c...`) |
| Concurrent `@claude` mentions overload | Concurrency groups cancel in-progress runs per PR |
| Dispatch to wrong workflow ref | `--ref main` hardcoded in CLI dispatcher |
| OIDC token obtained from non-main branch | Federation policies exact-match `@refs/heads/main` |
| Secrets accessed outside environment | `DECO_GITHUB_TOKEN` gated by `integration-tests` environment; `DECO_WORKFLOW_TRIGGER` by `test-trigger-is` |

## Open questions

1. **Bot identity**: Reviews are posted as `eng-dev-ecosystem-bot`. A custom GitHub App would give Claude a distinct identity. Separate effort.
2. **Rollout to other repos**: For each new repo, add a dispatcher workflow in the target repo, a repo-specific caller workflow in eng-dev-ecosystem, and a federation policy on the service principal.
