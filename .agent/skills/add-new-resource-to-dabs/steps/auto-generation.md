# Include resource in auto-generation

- Add the resource to [resources.go](./bundle/config/resources.go)
- Implement the resource in [bundle/config/resources/](./bundle/config/resources/)
    - Use one file per resource
    - URL should be the one from the Databricks Workspace
    - TODO: don't add unit tests as we prefer testserver over mocks
- Trigger auto-generations
    - `make schema`
        - You first should download the open API spec with `genkit get $(cat .codegen/_openapi_sha)`
            - If `genkit` isn't available, fall back to downloading `https://openapi.dev.databricks.com/OPENAPI_SHA/specs/all-internal.json`
        - Then run `DATABRICKS_OPENAPI_SPEC=/path/to/spec/file make schema`
    - `make schema-for-docs`
    - `make generate-validation`
    - `make generate-direct`
    - `make generate`?
- Add overrides to specs
    - TODO: on what conditions should we modify the below files?
    - annotations_openapi_overrides.yml
    - annotations.yml
