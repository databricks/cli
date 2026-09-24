#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 DIST VERSION {release|snapshot}" >&2
  exit 2
fi

DIST=$1
VERSION=$2
MODE=$3
case "$MODE" in
  release|snapshot) ;;
  *) echo "mode must be release or snapshot" >&2; exit 2 ;;
esac

if [[ ! -d "$DIST" ]]; then
  echo "release artifact directory does not exist: $DIST" >&2
  exit 1
fi

if [[ "$MODE" == release ]]; then
  archive_prefix="databricks_cli_${VERSION}_"
else
  archive_prefix="databricks_cli_"
fi
checksum="databricks_cli_${VERSION}_SHA256SUMS"
schema="$DIST/jsonschema.json"

expected=$(mktemp)
actual=$(mktemp)
trap 'rm -f "$expected" "$actual"' EXIT
for os in linux darwin windows; do
  for arch in amd64 arm64; do
    for format in zip tar.gz; do
      printf '%s%s_%s.%s\n' "$archive_prefix" "$os" "$arch" "$format" >> "$expected"
    done
  done
done
sort -o "$expected" "$expected"

if [[ ! -f "$DIST/$checksum" ]]; then
  echo "missing checksum manifest: $DIST/$checksum" >&2
  exit 1
fi
if [[ ! -f "$schema" ]]; then
  echo "missing bundle JSON schema: $schema" >&2
  exit 1
fi

find "$DIST" -maxdepth 1 -type f \( -name 'databricks_cli_*.zip' -o -name 'databricks_cli_*.tar.gz' \) -exec basename {} \; | sort > "$actual"
if ! diff -u "$expected" "$actual"; then
  echo "release archive set does not match the 3 OS x 2 architectures x 2 formats contract" >&2
  exit 1
fi

# The manifest is the authority for archive bytes. Check it from the directory
# containing the archives so paths in the manifest resolve exactly as published.
(
  cd "$DIST"
  sha256sum --check --strict "$checksum"
)

# The publishable set is exact. GoReleaser may additionally leave its metadata
# index, but no other files are accepted.
find "$DIST" -maxdepth 1 -type f -exec basename {} \; | sort > "$actual"
{
  cat "$expected"
  printf '%s\njsonschema.json\n' "$checksum"
} | sort > "$expected"
if ! diff -u "$expected" <(grep -v '^artifacts.json$' "$actual"); then
  echo "release artifact directory contains missing or extra files" >&2
  exit 1
fi

