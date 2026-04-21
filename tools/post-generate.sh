#!/bin/bash

set -euxo pipefail

# Ensure the SDK version is consistent with the OpenAPI SHA the CLI is generated from.
go test -timeout 240s -run TestConsistentDatabricksSdkVersion github.com/databricks/cli/internal/build

# Generate the bundle JSON schema.
make schema

# Fetch version tags (required for make schema-for-docs).
git fetch origin 'refs/tags/v*:refs/tags/v*'

make schema-for-docs

# Generate bundle validation code for enuma and required fields.
make generate-validation

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

# Generate PyDABs code.
make -C python codegen

# Fix whitespace issues in the generated code.
make wsfix
