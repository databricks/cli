# CLI Compatibility Manifest

`cli-compat.json` maps Databricks CLI versions to compatible AppKit and Agent Skills versions. The CLI uses this manifest to determine which template version to use for `apps init` and which skills version to use for `aitools install`.

## Manifest format

```json
{
  "0.296.0": { "appkit": "0.27.0", "skills": "0.1.5" },
  "0.290.0": { "appkit": "0.24.0", "skills": "0.1.4" },
  "0.280.0": { "appkit": "0.20.0", "skills": "0.1.0" }
}
```

Each key is a CLI version in semver format. Each entry defines a **range floor**: it applies to that CLI version and all versions above it, up to (but not including) the next entry. The manifest should be **sparse** — not every CLI version needs its own entry. Only add a new entry when a compatibility boundary changes.

For example, with the manifest above:
- CLI `0.285.0` → uses `0.280.0` entry (appkit `0.20.0`)
- CLI `0.293.0` → uses `0.290.0` entry (appkit `0.24.0`)
- CLI `0.300.0` → uses `0.296.0` entry (appkit `0.27.0`, highest versioned)
- CLI `0.0.0-dev+abc` → uses `0.296.0` entry (dev builds use the highest versioned entry)

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

The goal is to **keep the manifest sparse** — only add entries at compatibility boundaries. After each AppKit or Agent Skills release:

1. **Run evals** on the new AppKit version. If there is no regression, proceed.
2. **Open a PR** to update `cli-compat.json`. The change depends on the type of release:
   - **No breaking changes** (the new AppKit/skills version works with all existing CLI versions): update the existing highest versioned entry's appkit/skills values in-place. Do NOT add a new versioned key. All CLI versions in that range automatically pick up the new versions.
   - **Breaking changes** (the new AppKit templates require specific `apps init` features, or the new skills version requires CLI commands that older CLIs lack): add a new entry keyed to the **minimum CLI version** that supports the new features. Older entries keep their previous appkit/skills values so older CLI binaries stay compatible.

This process is manual for now but can be automated as part of the release workflow in the future. Use the `/bump-cli-compat` Claude Code skill to automate the update and PR creation.

## Validation

The manifest is validated by Go tests in `libs/clicompat/`:

```bash
go test ./libs/clicompat/... -run TestEmbeddedManifest -v
```

This checks: valid JSON, at least one entry, valid semver keys, valid semver entry values, and ascending key order.

## Pruning policy

Entries MUST NOT be removed from the manifest. Older CLI binaries use the lowest entry as their floor when the CLI version is older than all entries. Pruning it causes them to silently resolve to a newer entry that may require CLI features they lack. If the manifest grows too large, consider archiving very old entries to a separate file while keeping the oldest entry as a sentinel.

## Trust model

The live manifest is fetched over HTTPS from GitHub (`raw.githubusercontent.com`). The trust boundary is: TLS certificate validation + GitHub's access controls + write access to the `main` branch of `databricks/cli`. A compromised manifest can only steer clients to existing published tags (AppKit or skills); it cannot inject arbitrary code. The CLI binary always ships an embedded fallback manifest that limits exposure to cache-TTL windows (1 hour). The local cache (`~/.cache/databricks/compat-manifest.json`) is trust-on-disk: an attacker with user-level write access to the cache directory could swap in a malicious manifest pointing to different tags.
