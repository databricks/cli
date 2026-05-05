# CLI Compatibility Manifest

`cli-compat.json` maps Databricks CLI versions to compatible AppKit and Agent Skills versions. The CLI uses this manifest to determine which template version to use for `apps init` and which skills version to use for `aitools install`.

## Manifest format

```json
{
  "next": { "appkit": "0.24.0", "skills": "0.1.4" },
  "0.299.0": { "appkit": "0.24.0", "skills": "0.1.4" }
}
```

- Each key is a CLI version (`X.Y.Z`) or `"next"`.
- Each value specifies the compatible `appkit` and `skills` versions.
- `"next"` is used for CLI versions newer than any listed entry and for dev builds.

## How the CLI resolves versions

1. **Exact match** on CLI version → use that entry.
2. **No exact match**, between two entries → use the nearest lower version's entry.
3. **Newer than all entries** → use the highest versioned entry.
4. **Older than all entries** → use the lowest (oldest) entry.
5. **Dev builds** (`0.0.0-dev*`) → use `"next"`.

## Manifest sources (fallback chain)

At runtime, the CLI resolves the manifest from three sources:

1. **Local cache** (`~/.cache/databricks/compat-manifest.json`) — used if fresh (< 1 hour old).
2. **Remote fetch** from GitHub — used when cache is stale or missing. On success, the local cache is updated.
3. **Stale local cache** — if remote fetch fails but a previously cached file exists (even if expired), it is used as-is.
4. **Embedded manifest** — compiled into the binary via `go:embed`. Used as last resort when both remote and local cache fail.

## When to update

After each AppKit release:

1. **Run evals** on the new AppKit version. If there is no regression, proceed.
2. **Open a PR** to update `cli-compat.json`. The change depends on the type of release:
   - **No template changes** (just an AppKit/skills version bump): search & replace all version occurrences in the manifest and update `next`.
   - **Template changes that don't require new CLI features**: test the last 3 CLI versions with the new template and update matching entries.
   - **Template changes that require new CLI features**: add a new entry for the minimum CLI version that supports them; older entries keep pointing to the previous template version.

This process is manual for now but can be automated as part of the release workflow in the future. Use the `/bump-cli-compat` Claude Code skill to automate the update and PR creation.

## Validation

The manifest is validated by Go tests in `libs/depversions/`:

```bash
go test ./libs/depversions/... -run TestEmbeddedManifest -v
```

This checks: valid JSON, `"next"` key present, at least one versioned entry, valid semver keys, `next` versions >= all entries, and ascending key order.
