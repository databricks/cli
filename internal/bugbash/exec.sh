#!/usr/bin/env bash

set -euo pipefail

# Set the GitHub repository for the Databricks CLI.
export GH_REPO="databricks/cli"

# Synthesize the archive name for the snapshot build.
function cli_snapshot_archive() {
    name="databricks_cli"

    # Append OS
    case "$(uname -s)" in
    Linux)
        name="${name}_linux"
        ;;
    Darwin)
        name="${name}_darwin"
        ;;
    *)
        echo "Unknown operating system: $(uname -s)"
        return 1
        ;;
    esac

    # Append architecture
    case "$(uname -m)" in
    x86_64)
        name="${name}_amd64"
        ;;
    arm64|aarch64)
        name="${name}_arm64"
        ;;
    *)
        echo "Unknown architecture: $(uname -m)"
        return 1
        ;;
    esac

    echo "${name}.tar.gz"
}

BRANCH=$1
shift

# Default to main branch if branch is not specified.
if [ -z "$BRANCH" ]; then
  BRANCH=main
fi

if [ -z "$BRANCH" ]; then
  echo "Please specify which branch to bugbash..."
  exit 1
fi

# Check if the "gh" command is available.
if ! command -v gh &> /dev/null; then
  echo "The GitHub CLI (gh) is required to download the snapshot build."
  echo "Install and configure it with:"
  echo ""
  echo "  brew install gh"
  echo "  gh auth login"
  echo ""
  exit 1
fi

echo "Looking for a snapshot build of the Databricks CLI on branch $BRANCH..."

# Find last successful build on $BRANCH.
last_successful_run_id=$(
  gh run --repo databricks/cli list --branch "$BRANCH" --workflow release-build \
    --json 'databaseId,conclusion' --jq 'limit(1; .[] | select(.conclusion == "success")) | .databaseId'
)
if [ -z "$last_successful_run_id" ]; then
  echo "Unable to find last successful build of the release-build workflow for branch $BRANCH."
  exit 1
fi

# Download the release archives into ./bugbash relative to where the script is
# run. This is a stable location (not a temp dir) so `databricks ssh connect` can
# point its --releases-dir at the downloaded CLI archives. Use an absolute path
# so the reference stays valid if the user cd's elsewhere in the bugbash shell.
releases_dir="$PWD/bugbash"
rm -rf "$releases_dir"
mkdir -p "$releases_dir"

# Download the artifact.
echo "Downloading the snapshot build..."
gh run --repo databricks/cli download "$last_successful_run_id" --name cli --dir "$releases_dir"

# Extract the archive for this platform into a temporary directory.
archive=$(cli_snapshot_archive)
if [ ! -f "$releases_dir/$archive" ]; then
    echo "Archive not found: $archive"
    echo "Available archives:"
    ls "$releases_dir/"
    exit 1
fi

bin_dir=$(mktemp -d)
tar -xzf "$releases_dir/$archive" -C "$bin_dir"

# Make CLI available on $PATH.
chmod +x "$bin_dir/databricks"
export PATH="$bin_dir:$PATH"

# Set the prompt to indicate the bugbash environment and exec.
export PS1="(bugbash $BRANCH) \[\033[01;32m\]\u@\h\[\033[00m\]:\[\033[01;34m\]\w\[\033[00m\]\$ "

# Display completion instructions.
echo ""
echo "=================================================================="

if [[ ${BASH_VERSINFO[0]} -lt 5 ]]; then
    echo -en "\033[31m"
    echo "You have Bash version < 5 installed... completion won't work."
    echo -en "\033[0m"
    echo ""
    echo "Install it with:"
    echo ""
    echo "  brew install bash bash-completion"
    echo ""
    echo "=================================================================="
fi

echo ""
echo "To load completions in your current shell session:"
echo ""
echo "  source /opt/homebrew/etc/profile.d/bash_completion.sh"
echo "  source <(databricks completion bash)"
echo ""
echo "=================================================================="
echo ""
echo "To test 'databricks ssh connect', point --releases-dir at the downloaded"
echo "archives:"
echo ""
echo "  databricks ssh connect ... --releases-dir $releases_dir"
echo ""
echo "=================================================================="
echo ""

# Exec into a new shell.
# Note: don't use zsh because on macOS it _always_ overwrites PS1.
exec /usr/bin/env bash
