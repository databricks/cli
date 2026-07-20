# Appendix: Terraform Registry lag workaround

`./task generate-tf-schema` runs the codegen tool, which runs `terraform init`.
That pulls the provider from the **Terraform Registry** (`registry.terraform.io`), **not** from GitHub.
A freshly published GitHub release is not indexed by the registry immediately.
Indexing lags the GitHub release by anywhere from ~30 min to a few hours.

You only need this workaround if `./task generate-tf-schema` fails with:

```
Could not retrieve the list of available versions for provider
databricks/databricks: no available releases match the given constraints {version}
```

If codegen succeeded, ignore this file.

## Fix: point Terraform at a local filesystem mirror

Download the provider zip from the GitHub release (which always exists) into a filesystem-mirror layout, and set `TF_CLI_CONFIG_FILE` so `terraform init` resolves `databricks/databricks` from the mirror instead of the registry.
The env var propagates into the `go run .` that `generate-tf-schema` invokes.

Put the mirror in the repo's own `tmp/` directory, which is gitignored (the `/tmp/` entry in `.gitignore`).
Do NOT use the system `/tmp`: a sandboxed session may only be allowed to write inside the repo, and a mirror dir placed at a non-ignored path in the repo would pollute the acceptance test scan and get swept into `git add -A`.

```bash
# Download the provider zip from GitHub into a filesystem-mirror layout (repo-root ./tmp is gitignored).
MIRROR="$(pwd)/tmp/tf_mirror"
DIR="$MIRROR/registry.terraform.io/databricks/databricks"
mkdir -p "$DIR"
gh release download v{version} --repo databricks/terraform-provider-databricks \
  --pattern 'terraform-provider-databricks_{version}_linux_amd64.zip' \
  --dir "$DIR" --clobber

# Tell Terraform to resolve databricks/databricks from the mirror, everything else direct.
cat > "$MIRROR/cli.tfrc" <<EOF
provider_installation {
  filesystem_mirror {
    path    = "$MIRROR"
    include = ["registry.terraform.io/databricks/databricks"]
  }
  direct {
    exclude = ["registry.terraform.io/databricks/databricks"]
  }
}
EOF

# Re-run codegen against the mirror.
TF_CLI_CONFIG_FILE="$MIRROR/cli.tfrc" ./task generate-tf-schema
```

Adjust the `linux_amd64` arch in the download pattern for a different platform, e.g. `darwin_arm64`.
You can also just re-run `./task generate-tf-schema` without the mirror once the registry catches up, if you prefer.

Codegen still fetches the real GitHub SHA256 checksums for `root.go`, so nothing is faked.
The mirror only shortcuts the registry version lookup.

This workaround is only for codegen (Step 3).
The acceptance tests (Step 4) fetch the provider straight from the GitHub release via `acceptance/install_terraform.py`, so they do not depend on the registry and need no workaround.

Clean up the mirror when the bump is done:

```bash
rm -rf ./tmp/tf_mirror
```
