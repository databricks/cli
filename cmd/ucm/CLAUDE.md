# UCM (Unity Catalog Management) — Agent Guide

This file overrides and extends the repo-root `CLAUDE.md` for any work under
`cmd/ucm/`, `ucm/`, and `.github/workflows/upstream-sync.yml`. The root file
still applies for everything else (build commands, error handling, no
`os.Exit` outside main, etc.) — read it first.

## What ucm is

A sibling subcommand to `bundle` that brings DAB-style declarative management
to Unity Catalog: metastores, catalogs, schemas, volumes, external locations,
storage/service credentials, grants, tags, connections, plus the cloud
underlay (S3/ADLS/GCS, IAM/MI/SA, KMS) on AWS+Azure+GCP at parity.

Engine mirrors `bundle/` (~70–80% structural reuse): config loader, mutator
chain, terraform binary wrapper, lock/state, auth. Resource model and
converters are UC-specific and live in their own packages.

The full design is in `~/.claude/plans/you-are-a-data-twinkling-lighthouse.md`.

## Fork divergence rules — READ BEFORE EDITING

This repo is a fork of `databricks/cli`. A weekly upstream-sync workflow
(`.github/workflows/upstream-sync.yml`) merges `upstream/main`. Every change
to an upstream-owned file is potential merge conflict. So:

1. **Confine ucm code to ucm-owned paths**:
   - `cmd/ucm/**` — CLI wiring
   - `ucm/**` — engine (parallel sibling of `bundle/`)
   - `.github/workflows/upstream-sync.yml` — fork-only
2. **Touch upstream files only when unavoidable**. Currently the only allowed
   upstream edit is `cmd/cmd.go` (registers `ucm.New()` next to `bundle.New()`).
   Adding a new touchpoint requires a deliberate decision in the PR description.
3. **Never edit `bundle/**` from a ucm PR**. If you genuinely need a change in
   shared `libs/**`, extract it to ucm first; promote to upstream later.
4. **Don't import `bundle/**` from `ucm/**`**. Fork-and-adapt instead — the
   bundle package will evolve upstream and pinning to its internals will break
   on every sync. `libs/**` reuse is fine.

## Architecture

```
cmd/ucm/                 # cobra wiring, one file per verb
  ucm.go                 # New() — registers all verbs
  validate.go deploy.go plan.go destroy.go ...
  stubs.go               # placeholder for not-yet-implemented verbs

ucm/                     # engine (sibling of bundle/)
  bundle.go              # Ucm struct: Config, WorkspaceClient, AccountClient,
                         #   Terraform, Locker, StateFiler
  config/                # Root, Resources, Targets, Variables, mutator chain
  resources/             # Go struct per UC + cloud resource kind
  phases/                # load, initialize, build, deploy, destroy, plan, ...
  deploy/
    state.go state_pull.go state_push.go
    filer/               # StateFiler interface; workspace + s3/adls/gcs impls
    lock/                # forked from bundle/deploy/lock
    terraform/           # forked from bundle/deploy/terraform
      tfdyn/             # converter registry: ucm resource → tf json
  schema/                # generated jsonschema for ucm.yml
  generate/              # brownfield scan → ucm.yml + seed state
  templates/             # embed.FS, served by `ucm init`
  ci/                    # CI starter pipelines (GHA, GL, ADO, BB, Jenkins)
```

## Verbs

DAB-subset (parity): `validate`, `plan`, `deploy`, `destroy`, `summary`,
`init`, `generate`, `bind`, `schema`, `debug`, `diff`.
ucm-specific: `drift`, `import`, `policy-check`.
Dropped from DAB: `run`, `sync` (no runtime exec or source sync for UC).

All verbs are stubbed in `cmd/ucm/stubs.go` and return
`"ucm <verb> is not yet implemented"` — replace with real implementations one
verb at a time, each in its own `cmd/ucm/<verb>.go` file (mirror
`cmd/bundle/validate.go` for shape).

## Auth model

OAuth M2M with service principals only in v1 (no PAT, no U2M). The `Ucm`
struct holds both `WorkspaceClient` (workspace-scoped: grants, bindings,
catalogs/schemas/volumes/EL/SC/connections) AND `AccountClient` (account-scoped:
metastores, metastore assignments). Same SP must have account-admin scope.

Terraform inherits the same `DATABRICKS_HOST` / `DATABRICKS_CLIENT_ID` /
`DATABRICKS_CLIENT_SECRET` env vars that the `databricks` provider already
consumes. Cloud creds (AWS/Azure/GCP) come from environment — assume OIDC
federation in CI.

## State backend

Pluggable behind `StateFiler` interface (`ucm/deploy/filer/iface.go`):
- v1 impl: workspace files (DAB-parity)
- v2 impls: S3+DynamoDB / ADLS / GCS, selected via `ucm.yml > ucm.state.backend`

One `terraform.tfstate` per target (monolithic, not sharded). Terraform's
native reference graph orders resources — no cross-component orchestration.

## Topology

1 deployment = 1 account + 1 metastore + 1 workspace. Multi-region or
multi-workspace orgs run multiple deployments (`targets:` where shape is
similar, separate ucm.yml otherwise). The bootstrap deployment owns the
metastore; downstream workspace deployments reference its id read-only.

## Tag enforcement

`ucm/config/mutator/validate_tags.go` runs in `validate`/`plan`/`policy-check`.
Reads `resources.tag_validation_rules.*`, checks every securable matching
`securable_types` for required keys and (if `allowed_values` set) value
membership. Emits `error`-level diagnostics. No dependency on UC server-side
tag policy.

## Build / test (UCM-specific shortcuts)

The repo-root `CLAUDE.md` has the canonical build commands. UCM-specific
gotchas in this environment:

```bash
# Corporate /etc/hosts blocks proxy.golang.org and toolchain may not match go.mod.
# Use these env vars for any go command:
GOPROXY=direct GOTOOLCHAIN=local GOSUMDB=off go build ./...
GOPROXY=direct GOTOOLCHAIN=local GOSUMDB=off go test ./cmd/ucm/... ./ucm/...

# git-lfs is required (charmbracelet/huh transitive dep). Install once:
brew install git-lfs   # do NOT run `git lfs install` — pre-push hook conflict

# Smoke test the binary:
./databricks ucm --help
./databricks ucm validate   # expect: "ucm validate is not yet implemented"
```

When implementing a verb, add unit tests next to the file (`cmd/ucm/<verb>_test.go`)
and mutator/converter tests under `ucm/config/mutator/*_test.go` and
`ucm/deploy/terraform/tfdyn/*_test.go` following bundle's patterns.

## Git workflow (gitops-style — required for UCM work)

The upstream `databricks/cli` `CLAUDE.md` doesn't prescribe an issue→branch→PR
lifecycle. UCM does — adapted from the prior Python implementation's gitops
agent. Use this for every non-trivial UCM change:

### 1. Create a tracking issue first

```bash
gh issue create \
  --title "<short imperative summary>" \
  --body "<context, acceptance criteria, links>" \
  --label "ucm,<area-label>"
```

Area labels: `area/cmd`, `area/config`, `area/mutator`, `area/terraform`,
`area/state`, `area/cloud-aws`, `area/cloud-azure`, `area/cloud-gcp`,
`area/templates`, `area/ci`, `area/docs`.

Type labels (one): `feat`, `fix`, `chore`, `refactor`, `test`, `docs`.

### 2. Branch naming

`<type>/<issue-number>-<kebab-summary>` — examples:
- `feat/12-validate-tags-mutator`
- `fix/18-state-filer-lock-timeout`
- `chore/3-upstream-sync-2026-04-27`

One issue per branch. One branch per PR. Never push directly to `main`.

### 3. Commit messages

`<Type> #<issue>: <subject>` — examples:
- `Feat #12: Add validate_tags mutator`
- `Fix #18: Retry workspace-files lock acquisition with backoff`

Body explains *why*. Never amend pushed commits. Never `--no-verify`.

### 4. Pre-push CI gate (run locally)

```bash
GOPROXY=direct GOTOOLCHAIN=local GOSUMDB=off go build ./...
GOPROXY=direct GOTOOLCHAIN=local GOSUMDB=off go test ./cmd/ucm/... ./ucm/...
GOPROXY=direct GOTOOLCHAIN=local GOSUMDB=off go vet ./cmd/ucm/... ./ucm/...
```

Block the push if any step fails. Fix root cause — don't bypass.

### 5. PR template

```markdown
Closes #<N>

## Summary
- <bullet 1>
- <bullet 2>

## Why
<one paragraph: motivation, constraint, or upstream incident>

## Test plan
- [ ] `go build ./...`
- [ ] `go test ./cmd/ucm/... ./ucm/...`
- [ ] Manual: <smoke test command>
- [ ] If touching upstream files: list each file and justify
```

PR labels mirror issue labels. Request review only after CI is green.

### 6. Never

- Push to `main` directly.
- Force-push (use `--force-with-lease` only on your own feature branch).
- Skip hooks (`--no-verify`, `--no-gpg-sign`).
- Squash-merge across multiple issues — one PR closes one issue.
- Edit `bundle/**` or `libs/**` from a ucm-labeled PR.

## Common mistakes (UCM-specific, on top of root CLAUDE.md)

- Importing from `bundle/**` instead of forking the relevant pieces into
  `ucm/**`. The bundle package is upstream and will diverge on every sync.
- Adding new edits to `cmd/cmd.go` (or any other upstream file) without
  flagging it as a fork-divergence cost in the PR.
- Hand-writing TF JSON in converters — use `libs/dyn` to build values, then
  `tfjson` write helpers, exactly as `bundle/deploy/terraform/tfdyn` does.
- Adding identity (groups/SCIM/users/SPs) resources. Out of scope for v1 —
  ucm references existing principals by name/id only.
- Adding a `run` or `sync` verb. Explicitly dropped from DAB scope.
- Treating "policy-check" as a slower `validate`. policy-check runs *only*
  the validation mutators (cheap pre-commit hook target); `validate` runs the
  full chain.
