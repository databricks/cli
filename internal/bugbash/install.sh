#!/usr/bin/env bash
# Install the latest demo-lakebox snapshot of the databricks CLI to
# ~/.databricks-snapshot/bin/databricks. Re-run any time you want to
# pick up new commits on the branch — the latest successful CI
# release-build is always what you get.
#
# The install dir is NOT added to PATH by default to avoid shadowing
# any system `databricks` binary. The script prints activation options
# at the end; pick whichever fits your workflow.

set -euo pipefail

REPO="databricks/cli"
BRANCH="demo-lakebox"
INSTALL_DIR="$HOME/.databricks-snapshot/bin"

OS="$(uname -s | tr 'A-Z' 'a-z')"
ARCH="$(uname -m)"; [[ $ARCH == x86_64 ]] && ARCH=amd64
[[ $ARCH == aarch64 ]] && ARCH=arm64
case "$OS" in linux|darwin) ;; *) echo "unsupported OS: $OS" >&2; exit 1 ;; esac
case "$ARCH" in amd64|arm64) ;; *) echo "unsupported arch: $ARCH" >&2; exit 1 ;; esac

for tool in gh jq curl; do
  command -v "$tool" >/dev/null 2>&1 || { echo "$tool not installed (try: brew install $tool)" >&2; exit 1; }
done

mkdir -p "$INSTALL_DIR"

echo "→ looking up latest successful release-build run on $REPO @ $BRANCH …"
rid=$(gh run list -b "$BRANCH" -w release-build -R "$REPO" \
      --json databaseId,conclusion --limit 5 \
      | jq -r 'limit(1; .[] | select(.conclusion=="success")) | .databaseId')
[[ -n $rid && $rid != null ]] || { echo "no successful release-build run found on $BRANCH" >&2; exit 1; }

tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
echo "→ downloading artifact from run $rid …"
( cd "$tmp" && gh run download "$rid" -R "$REPO" -n cli >/dev/null )

tar="$tmp/databricks_cli_${OS}_${ARCH}.tar.gz"
[[ -f $tar ]] || { echo "expected platform tarball not in artifact: $tar" >&2; exit 1; }
( cd "$tmp" && tar -xzf "$tar" )
install -m 755 "$tmp/databricks" "$INSTALL_DIR/databricks"

version=$("$INSTALL_DIR/databricks" --version)
echo
echo "✓ installed $version to $INSTALL_DIR/databricks (CI run $rid)"
echo
echo "The install dir is intentionally NOT on PATH. Activate it one of three ways:"
echo
echo "  1. Just this shell, alongside your normal databricks:"
echo "       alias databricks=\"$INSTALL_DIR/databricks\""
echo
echo "  2. All future shells (replaces system databricks for new terminals):"
echo "       echo 'export PATH=\"\$HOME/.databricks-snapshot/bin:\$PATH\"' >> ~/.zshrc"
echo "       source ~/.zshrc"
echo
echo "  3. Without shadowing — invoke by full path or a distinct name:"
echo "       $INSTALL_DIR/databricks lakebox list"
echo "       ln -s $INSTALL_DIR/databricks ~/.local/bin/databricks-snapshot   # then run \`databricks-snapshot lakebox …\`"
