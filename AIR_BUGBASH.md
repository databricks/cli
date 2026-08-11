# `air` CLI (Go port) bug bash

The `air` CLI has been ported from Python to Go, and now ships inside the
Databricks CLI as `databricks experimental air`. Same commands you know — `run`,
`get`, `list`, `logs`, `cancel`, `register-image` — reimplemented. We want to know
where the port diverges from what you'd expect.

## Prerequisites

- Access to a workspace with **serverless GPU compute** enabled, e.g.
  go/e2-dogfood, go/azure-dogfood, go/df1
- Go 1.26.x and git

## Setup

```sh
# 1. Clone the Databricks CLI and build (~2 min). `air` is on main — no branch needed.
git clone https://github.com/databricks/cli.git ~/databricks-cli
cd ~/databricks-cli && ./task build

# 2. Install as `databricks` on your PATH.
mkdir -p ~/bin && cp ./cli ~/bin/databricks && export PATH="$HOME/bin:$PATH"

# 3. Authenticate (opens browser).
databricks auth login --profile air-bugbash
export DATABRICKS_CONFIG_PROFILE=air-bugbash

# 4. Test installation.
databricks experimental air --help
```

Optional: `docker login <registry>` if you want to test `register-image` with a
private image.

Handy alias, used throughout this doc: `alias air='databricks experimental air'`

### A minimal config to start from

```yaml
# ~/train.yaml — the sleep keeps the run alive long enough to test logs and cancel
experiment_name: my-air-bugbash-run
compute: {accelerator_type: GPU_1xA10, num_accelerators: 1}
command: echo "hello AIR"; sleep 300
```

Then `air run -f ~/train.yaml`. Start with `--dry-run` if you want to check a config
before spending GPU time on it.

---

## CUJs

**When you pick up a CUJ, add your name to the feedback table at the bottom.**

Throughout: most commands take `-o json` as well as the default text output. Please
try both — a broken JSON envelope is as much a bug as broken text, and the error
paths matter most (stdout should stay valid JSON even when something fails).

### CUJ 1: Submit a run and watch it

Start with the minimal config above and just `air run -f ~/train.yaml`. Then start
layering on the flags: `--dry-run` to validate without submitting (it makes no
workspace calls at all, so it should work even with credentials unset),
`--override compute.num_accelerators=8` to patch a field from the command line
without editing YAML (repeatable), and `--idempotency-key <something>` to make a
resubmission return the *existing* run instead of starting a second one — run it
twice with the same key and confirm you get one run, not two.

The interesting one is `--watch`, which submits and then streams logs until the run
finishes. Check that the exit code reflects the run's actual outcome, and that
Ctrl-C partway through exits cleanly rather than hanging or dumping a stack trace.

Then try to break the config. Bad values should be rejected clearly and early, with
a message that tells you what to fix: 3 accelerators on a `GPU_8xH100` (not a
multiple of 8), the same name in both `env_variables` and `secrets`, a missing
`command`, `max_retries: -1`, `timeout_minutes: 0`, an `mlflow_experiment_directory`
that doesn't start with `/Workspace`, both `usage_policy_name` and
`usage_policy_id` set at once, or `environment.version` alongside a
`dependencies:` file path (the file carries its own version, so setting both is
contradictory).

### CUJ 2: Ship real code with `code_source`

Point a run at a local directory and confirm the code actually arrives:

```yaml
code_source:
  type: snapshot
  snapshot:
    root_path: .
    include_paths: [train.py]     # omit to include everything
```

Worth poking at: `include_paths` should genuinely exclude what you left out (drop a
large junk file in the directory and confirm it isn't uploaded), `git: {commit: <sha>}`
should pin the snapshot to that revision rather than archiving your working tree,
and `remote_volume: /Volumes/...` should upload there instead of the default
location. Resubmitting at the same pinned commit is supposed to reuse a cached
tarball — see whether the second submit is noticeably faster.

Also try dependencies both ways: an inline list (`dependencies: [numpy, torch==2.3.0]`)
and a path to a `requirements.yaml`. Both should end up installed in the run.

### CUJ 3: Multi-node

Anything with `num_accelerators` spanning more than one node — e.g. 16 on
`GPU_8xH100` — exercises a lot of code that single-node runs never touch. Submit
one, then use `air get` and `air logs --node N` against it, and cross-check the
Jobs page and MLflow page in the UI against what the CLI reports.

### CUJ 4: `list`, `get`, `logs`, `cancel`

Test out the 4 core commands with multiple active runs. Specific flags to test out:

**`air list`** — interactive table in a terminal; prints once when piped or under `NO_COLOR=1`.
- Navigate with `j`/`k` or arrows, `g`/`G` for top/bottom, left/right to page, `enter` to open a run, `q` to quit. Page back through older runs and watch for responsiveness.
- `--limit N` — caps the rows, and switches off the TUI.
- `--all-status` — include finished runs, not just active ones.
- `--all-users` — everyone's runs, not just yours.
- `--filter KEY=VALUE` — repeatable and ANDed. Keys: `experiment` (glob, e.g. `'foo*'`), `accelerator_type` (substring, e.g. `H100`), `num_accelerators` (exact), `user` (email).
- Run it twice — the second should be faster, since finished runs cache to disk.

**`air get <run-id>`** — one run in detail.
- Cross-check status, duration, accelerators, and environment against the Jobs/MLflow UI.
- Click the run ID and MLflow hyperlinks.
- On a sweep (`foreach`) run, expect a per-iteration breakdown instead of the single-run view.

**`air logs <run-id>`** — new backend, so give this the most attention.
- No flags: stream a run producing lots of output and see whether it keeps up.
- `--lines N` — tail the last N lines of a finished run.
- `--minutes N` — only the last N minutes.
- `--node N` — isolate one node of a multi-node run.
- `--retry N` — read an earlier attempt of a retried run.
- `--download-to <dir>` — brand new; pulls every node's logs to disk in parallel, one file per node. Check the files are complete and in order, and that one unavailable node warns you rather than silently returning a truncated log.
- Rejections should be explicit, not silent: `--download-to` with `--lines`/`--minutes`, `--lines` with `--minutes`, `--node 5` on a 2-node run.
- Edge cases: a run that produced no logs, and Ctrl-C mid-stream and mid-download.

**`air cancel <run-id>`** — prompts first.
- `-y` skips the confirmation.
- Several run IDs in one call.
- `--all` — every one of *your* active runs, across all pages, and nobody else's.
- Errors: IDs combined with `--all`, and neither one given.

### CUJ 5: Custom images — `register-image`

Walk the whole path once with a private image and judge how it reads:

```sh
docker login <registry>
air register-image <registry>/<repo>:<tag>   # prints digest + a config snippet
# paste that snippet into ~/train.yaml, then:
air run -f ~/train.yaml --dry-run
```

Credentials should be picked up from your local Docker config with no flags to pass,
the printed snippet should paste in and validate as-is, and re-registering should
report cached-vs-updated truthfully. Failures should say whether retrying helps, and
the replication wait should show progress rather than looking like a hang.

> **Known gap:** `air run` ignores `environment.docker_image` at submit — that DCS
> wiring is on an unmerged branch, so don't file it. Do file anything misleading
> along the way, like a snippet the config parser rejects or a success message for
> an image that isn't ready.

### CUJ 6: Chain them together

The seams between commands are where bugs hide. A few chains worth walking:

- `air run` → `air get` → `air logs` → `air cancel` → `air get` again and confirm
  it shows as cancelled.
- `air run --watch` in one terminal, `air cancel <id> -y` in another. The watching
  process should notice and exit cleanly with a sensible code.
- Submit a few runs → find them in `air list` → narrow with `--filter` →
  `air cancel --all -y` → confirm `air list` is empty but `--all-status` still
  shows them.
- Run the same run ID through every command with `-o json` and pipe each to `jq`.

---

## Known issues — please don't file these

- `air run` doesn't use a registered Docker image yet (pending proto change).
- `air logs --review` reports "not implemented yet" (hidden flag).
- Interactive `air list` caps its page size to the terminal height, so a short
  window shows only a few runs. Fix exists, not on this build.
- `air run -h config` prints ordinary command help — the inline config-schema
  reference is still on an unmerged branch (`air-config-help`, needs a rebase).
- `--tag-policy` on `register-image` is deprecated and ignored.

## Filing bugs

Include the command verbatim, the config YAML, the output in both text and
`-o json`, the run ID, and `git rev-parse HEAD` from your checkout.

## BugBash feedback

| CUJ | Name | Findings |
| --- | --- | --- |
| 1: Submit and watch | | |
| 2: code_source | | |
| 3: Multi-node | | |
| 4: list/get/logs/cancel | | |
| 5: register-image | | |
| 6: Chains | | |
