## docs-autogen

1. Install [Golang](https://go.dev/doc/install)
2. Run `go mod download` from the repo root
3. Run `make docs` from the repo
4. See generated documents in `./bundle/internal/docs/output` directory
5. To change descriptions update content in `./bundle/internal/schema/annotations.yml` or `./bundle/internal/schema/annotations_openapi_overrides.yml` and re-run `make docs`

For simpler usage run it together with copy command to move resulting files to local `docs` repo. Note that it will overwrite any local changes in affected files. Example:

```
make docs && cp bundle/internal/docs/output/*.md ../docs/source/dev-tools/bundles
```

To change file names or file headers update them in `main.go` file in this directory

### Annotation file structure

```yaml
"<root-type-name>":
  "<property-name>":
    description: Description of the property, only plain text is supported
    markdown_description: Description with markdown support, if defined it will override the value in docs and in JSON-schema
    markdown_examples: Custom block for any example, in free form, Markdown is supported
    title: JSON-schema title, not used in docs
    default: Default value of the property, not used in docs
    enum: Possible values of enum-type, not used in docs
```

Descriptions with `PLACEHOLDER` value are not displayed in docs and JSON-schema

All relative links like `[_](/dev-tools/bundles/settings.md#cluster_id)` are kept as is in docs but converted to absolute links in JSON schema

### Example annotation

```yaml
github.com/databricks/cli/bundle/config.Bundle:
  "cluster_id":
    "description": |-
      The ID of a cluster to use to run the bundle.
    "markdown_description": |-
      The ID of a cluster to use to run the bundle. See [_](/dev-tools/bundles/settings.md#cluster_id).
  "compute_id":
    "description": |-
      PLACEHOLDER
  "databricks_cli_version":
    "description": |-
      The Databricks CLI version to use for the bundle.
    "markdown_description": |-
      The Databricks CLI version to use for the bundle. See [_](/dev-tools/bundles/settings.md#databricks_cli_version).
  "deployment":
    "description": |-
      The definition of the bundle deployment
    "markdown_description": |-
      The definition of the bundle deployment. For supported attributes, see [_](#deployment) and [_](/dev-tools/bundles/deployment-modes.md).
  "git":
    "description": |-
      The Git version control details that are associated with your bundle.
    "markdown_description": |-
      The Git version control details that are associated with your bundle. For supported attributes, see [_](#git) and [_](/dev-tools/bundles/settings.md#git).
  "name":
    "description": |-
      The name of the bundle.
  "uuid":
    "description": |-
      PLACEHOLDER
```

### TODO

Add file watcher to track changes in the annotation files and re-run `make docs` script automtically
