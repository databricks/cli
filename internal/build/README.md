# CLI Compatibility Manifest

`cli-compat.json` maps Databricks CLI versions to compatible AppKit and Agent Skills versions. The CLI uses this manifest to determine which template version to use for `apps init` and which skills version to use for `aitools install`.

Raw skills track the latest published skills release for the newest CLI range. Plugin installs use the agent's marketplace commands without a version pin; the manifest's skills settings only control raw-file delivery.

## Manifest format

```json
{
  "1.22.0": { "appkit": "0.84.0", "skills": "latest" },
  "1.20.0": { "appkit": "0.84.0", "skills": "0.2.30" },
  "1.0.0": { "appkit": "0.76.1", "skills": "0.2.10" },
  "0.299.2": { "appkit": "0.24.0", "skills": "0.1.5" }
}
```

Each key is a CLI version in semver format. Each entry defines a **range floor**: it applies to that CLI version and all versions above it, up to (but not including) the next entry. The manifest should be **sparse** — not every CLI version needs its own entry. Only add a new entry when a compatibility boundary changes.

`skills` is an exact pin for its CLI range or `"latest"` to install the skills repository's latest published GitHub release. When the latest-release lookup fails or the selected skills manifest is missing, ranges tracking latest use `AgentSkillsFallbackVersion` from `internal/build/clicompat.go`. Pinned ranges use their embedded manifest's skills version as the fallback. Changing either fallback takes effect in the next CLI build. Released binaries keep the fallback they shipped with.

An entry can also set `"auto_install": true` to turn on the automatic installation of Databricks AI tools on the first run of a new CLI version in its range (`aitools auto-install`). It is off when absent. A change reaches a machine on its first run of a CLI version after the change; machines that resolve to a cached or embedded manifest keep the value they have.

An entry can also set `"skills_hints": true` to turn on the hints that point agents at the skills commands. It is off when absent and needs `DATABRICKS_SKILLS_PREVIEW` as well. `DATABRICKS_SKILLS_HINTS`, when set, overrides the entry in both directions. Unlike `auto_install`, this field is never fetched, because hints are resolved while help is printed. It is read only from the local cache while that cache is younger than one hour. Without a fresh cache, hints are off: the embedded manifest is not used, so a machine with no cache, a stale cache, `DATABRICKS_CACHE_ENABLED=false` or `DATABRICKS_FORCE_EMBEDDED_COMPAT=true` keeps hints off, and it turns on only after a command that fetches the manifest has written a fresh cache. A machine whose cache goes stale turns hints off again until then. Since the embedded value is never read, a remote `false` turns hints off on every CLI that refreshes its cache, released ones included. A CLI from before this field re-encodes the cache when it refreshes it and drops the field, so on this CLI hints read as off until it refreshes the cache itself.

This skills fallback only applies to raw-file delivery. Version checks omit update availability when latest is unknown, and plugin operations don't substitute the fallback version.

Raw-skills updates and reinstalls reject automatic downgrades, including with `--force`: the fallback must be at least as new as the installed release. Existing installs without a comparable release require a successful latest-release fetch. Explicit `DATABRICKS_SKILLS_REF` overrides and compatibility pins can still select an older release, except in automatic installation: it keeps an installed release newer than the pinned one, which `databricks aitools update` still installs.

CLIs before `1.6.0` reject the whole remote manifest when any entry uses `"latest"`, even an entry they never resolve to. They use a valid cached manifest or their embedded manifest without returning a parse error. These CLIs stop receiving remote compatibility updates for both AppKit and skills.

Latest tracking starts at the CLI `1.22.0` range. CLI `1.20.x` and `1.21.x` stay pinned.

For example, with the manifest above:
- CLI `0.299.5` → uses `0.299.2` entry (appkit `0.24.0`)
- CLI `1.19.0` → uses `1.0.0` entry (appkit `0.76.1`)
- CLI `1.21.0` → uses `1.20.0` entry (appkit `0.84.0`, skills `0.2.30`)
- CLI `1.22.0` → uses `1.22.0` entry (appkit `0.84.0`, latest skills with compiled fallback `0.2.30`)
- CLI `0.0.0-dev+abc` → uses `1.22.0` entry (dev builds use the highest versioned entry)

## How the CLI resolves versions

1. **Dev builds** (`0.0.0-dev*`) → use the highest versioned entry.
2. **Exact match** on CLI version → use that entry.
3. **No exact match**, between two entries → use the nearest lower version's entry.
4. **Newer than all entries** → use the highest versioned entry.
5. **Older than all entries** → use the lowest (oldest) entry.

## Manifest sources (fallback chain)

At runtime, the CLI resolves the manifest from four sources:

1. **Local cache** (`~/.cache/databricks/compat-manifest.json`) — used if fresh (< 1 hour old).
2. **Remote fetch** from GitHub — used when cache is stale or missing. On success, the local cache is updated.
3. **Stale local cache** — if remote fetch fails but a previously cached file exists (even if expired), it is used as-is.
4. **Embedded manifest** — compiled into the binary via `go:embed`. Used as last resort when both remote and local cache fail.

Set `DATABRICKS_FORCE_EMBEDDED_COMPAT=true` to skip all tiers and use only the embedded manifest. This is useful for local development when testing with a locally compiled binary that has a modified `cli-compat.json`.

## When to update

The goal is to **keep the manifest sparse** — only add entries at compatibility boundaries. Routine skills releases do not require a manifest update: keep `"skills": "latest"` on the newest range.

When updating AppKit or recording a tested skills fallback:

1. **Run evals** on the new AppKit version. If there is no regression, proceed.
2. **Open a PR** to update `cli-compat.json`. The change depends on the type of release:
   - **No breaking changes**: update the highest entry's `appkit` in-place, keeping `"skills": "latest"`. To record a tested skills fallback, update `AgentSkillsFallbackVersion` in `internal/build/clicompat.go`. Do NOT add a new versioned key.
   - **Breaking changes** (the new AppKit templates require specific `apps init` features, or the new skills version requires CLI commands that older CLIs lack): add a new entry keyed to the **minimum CLI version** that supports the new features. Set `"skills": "latest"` on the new entry and update the compiled skills fallback to a tested release. If skills compatibility changed, replace `"latest"` in the affected older ranges with the last compatible release. Preserve unrelated AppKit and skills compatibility boundaries. If the previous highest entry sets `"auto_install": true`, copy it to the new entry: a new entry without it turns auto-install off for every newer CLI. Do the same for `"skills_hints": true`, which turns skill hints on for its range.

This process is manual for now but can be automated as part of the release workflow in the future. Use the `/bump-cli-compat` Claude Code skill to automate the update and PR creation.

## Validation

The manifest is validated by Go tests in `libs/clicompat/`:

```bash
go test ./libs/clicompat/... -run TestEmbeddedManifest -v
```

This checks: valid JSON, at least one entry, valid semver keys and AppKit versions, concrete skills versions or `"latest"`, a concrete compiled skills fallback, and latest tracking on the newest CLI range.

## Pruning policy

Entries MUST NOT be removed from the manifest. Older CLI binaries use the lowest entry as their floor when the CLI version is older than all entries. Pruning it causes them to silently resolve to a newer entry that may require CLI features they lack. If the manifest grows too large, consider archiving very old entries to a separate file while keeping the oldest entry as a sentinel.

## Bundled skills snapshot

The embedded fallback for skills lives in `libs/skills/snapshot`. Only its empty `manifest.json` is committed. The release build fills it, so release binaries carry the skills release that is current when they are built. No skill files are committed.

`internal/build/skillssnapshot` does the fill. The `before` hooks in `.goreleaser.yaml` run it before the binaries compile, so every release build from GoReleaser runs it. Development builds (`./task build`, `./task snapshot`) do not, and leave the working tree unchanged.

- It resolves the latest skills release (or the tag passed with `-ref`, for example `go run ./internal/build/skillssnapshot -ref v0.2.54`), downloads its `manifest.json`, applies the manifest path checks of `skills.CheckManifest`, and downloads the catalog rules and every skill file into a staging folder beside the snapshot.
- A file that cannot be downloaded, a manifest that names a path outside the release, or a file that `go:embed` would skip (a path element starting with `.` or `_`) fails the command. The snapshot is replaced only after every file is in place and the staged release reads back through the bundled source, so a failed run leaves the snapshot as it was.
- To check locally, run the command from the repository root, then build. Restore the committed placeholder afterwards with `git checkout -- libs/skills/snapshot` and `git clean -fd libs/skills/snapshot`.
- The latest-release lookup is unauthenticated, like the installer's, so it shares the GitHub API rate limit of the runner's IP address.

## Trust model

The live manifest is fetched over HTTPS from GitHub (`raw.githubusercontent.com`). The trust boundary is: TLS certificate validation + GitHub's access controls + write access to the `main` branch of `databricks/cli`. A compromised manifest can steer clients to existing published tags (AppKit or skills), and through `auto_install` make them install those published skills and plugins without a command from the user; it cannot inject arbitrary code. The CLI binary always ships an embedded fallback manifest that limits exposure to cache-TTL windows (1 hour). The local cache (`~/.cache/databricks/compat-manifest.json`) is trust-on-disk: an attacker with user-level write access to the cache directory could swap in a malicious manifest pointing to different tags.
