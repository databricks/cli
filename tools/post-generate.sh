# Ensure the SDK version is consistent with the OpenAPI SHA the CLI is generated from.
go test -timeout 240s -run TestConsistentDatabricksSdkVersion github.com/databricks/cli/internal/build

# Generate the bundle JSON schema.
make schema

# Generate bundle validation code for enuma and required fields.
make generate-validation

# Remove the next-changelog.yml workflow.
rm .github/workflows/next-changelog.yml

# Move the tagging.py file to the internal/genkit/tagging.py file. We do this to avoid
# cluttering the root directory.
mv tagging.py internal/genkit/tagging.py

# Update the tagging.yml workflow to use the new tagging.py file location.
sed -i 's|python tagging.py|python internal/genkit/tagging.py|g' .github/workflows/tagging.yml
./tools/yamlfmt .github/workflows/tagging.yml

# Generate PyDABs code.
make -C experimental/python codegen
