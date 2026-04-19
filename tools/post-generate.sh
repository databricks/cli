#!/bin/bash

set -euxo pipefail

# This script runs inside `task generate:commands` via .codegen.json's
# `toolchain.post_generate`. It post-processes files that genkit just wrote
# (generated commands in cmd/, tagging.py, tagging.yml). Generators that read
# bundle config (schema, schema-for-docs, validation, docs) and Python codegen
# live as separate Taskfile tasks and run from the `generate` aggregator — go.mod
# being in their sources invalidates their caches when genkit bumps the SDK.

# Ensure the SDK version is consistent with the OpenAPI SHA the CLI is generated from.
go test -timeout 240s -run TestConsistentDatabricksSdkVersion github.com/databricks/cli/internal/build

# Remove the next-changelog.yml workflow.
rm .github/workflows/next-changelog.yml

# Move the tagging.py file and its lock file to internal/genkit/. We do this to
# avoid cluttering the root directory. The lock file must stay next to tagging.py
# for `uv run --locked` to work in the tagging workflow.
mv tagging.py internal/genkit/tagging.py
mv tagging.py.lock internal/genkit/tagging.py.lock

# Update the tagging.yml workflow to use the new tagging.py file location.
# The genkit generates "uv run --locked tagging.py", we need to rewrite it
# to point at the moved location.
if [[ "$(uname)" == "Darwin" ]]; then
    # macOS (BSD sed) requires empty string after -i
    sed -i '' 's|tagging.py|internal/genkit/tagging.py|g' .github/workflows/tagging.yml
else
    # Linux (GNU sed)
    sed -i 's|tagging.py|internal/genkit/tagging.py|g' .github/workflows/tagging.yml
fi
go tool -modfile=tools/go.mod yamlfmt .github/workflows/tagging.yml

# Fix whitespace issues in the generated code.
./tools/validate_whitespace.py --fix
