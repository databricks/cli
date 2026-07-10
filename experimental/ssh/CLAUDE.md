# Working on `experimental/ssh`

This file provides guidance to AI assistants working on the SSH tunnel. See the
repository root `CLAUDE.md` for project-wide rules.

## Testing an `ssh connect` change against real compute

The binary uploaded to the cluster **is** the server: `ssh connect` uploads the
local `databricks` CLI and runs it in server mode on the compute. So a fix only
takes effect if the binary you upload actually contains it. Several ways to
silently test stale code — verify each before trusting a result:

**RULE: Confirm the binary you run is built from your current checkout.** When a
fix lives on a branch (or a git worktree), building or running `./cli` from a
different directory compiles unrelated code. The connection still succeeds and
looks correct, but the server runs the wrong binary. Before connecting:

```sh
go build -o ./cli . && ./cli version
git rev-parse --short HEAD   # the 0.0.0-dev+<sha> suffix must match this
```

If the version suffix doesn't match `HEAD`, you are running old code — stop and
rebuild from the right directory.

**RULE: For `--releases-dir`, build with `./task snapshot-release`, not `./task build`.**
`./task build` produces only `./cli`; it does not populate `./dist`. The tunnel's
`--releases-dir` loader expects `dist/databricks_cli_linux_{amd64,arm64}.zip`
(see `getLocalRelease` in `internal/client/releases.go`). With a stale or empty
`./dist`, the upload finds nothing new and the cluster reuses an older binary.
Confirm the archive carries your change before uploading, e.g.:

```sh
unzip -p ./dist/databricks_cli_linux_amd64.zip databricks | strings | grep <a-string-your-change-adds>
```

**RULE: Dev/snapshot uploads are skipped when the versioned workspace directory already exists.**
`uploadReleases` (`internal/client/releases.go`) skips the upload when the binary
is already present at the versioned workspace path. Dev builds keep the same
version string (`0.0.0-dev+<commit>`) across rebuilds, so a rebuilt server binary
is silently **not** re-uploaded and the cluster runs the stale one. Before
re-verifying a rebuilt binary, delete the versioned directory and connect with a
fresh `--name`:

```sh
VER=$(./cli version | grep -o '0.0.0-dev+[a-f0-9]*')
databricks workspace delete --recursive \
  "/Workspace/Users/<you>/.databricks/ssh-tunnel/$VER"
```

## Server vs. session

The server process and the interactive login session run as the same OS user with
the same `$HOME` on serverless compute (CPU and GPU alike). When a session-level
symptom (wrong `PATH`, a file not seeded, etc.) appears only on one compute type,
check the binary version first — it is far more often a stale upload than a
genuine per-compute difference.
