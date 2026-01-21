# NEXT CHANGELOG

## Release v0.284.0

### Notable Changes

### CLI

### Bundles
* Pass additional Azure DevOps `SYSTEM_*` environment variables to Terraform for OIDC authentication: `SYSTEM_COLLECTIONURI`, `SYSTEM_DEFINITIONID`, `SYSTEM_HOSTTYPE`, `SYSTEM_JOBID`, `SYSTEM_TEAMPROJECT` ([#4318](https://github.com/databricks/cli/pull/4318))
* Add support for valueFrom property (similar to app.yaml) inside Apps config field in bundle configuration ([#4297](https://github.com/databricks/cli/pull/4297))
* engine/direct: Support bind & unbind. ([#4279](https://github.com/databricks/cli/pull/4279))
* engine/direct: Ignore changes between nulls and empty slices/maps (([#4313](https://github.com/databricks/cli/pull/4313)))
* engine/direct: Ignore changes between nulls and empty structs (([#4338](https://github.com/databricks/cli/pull/4338)))
* On terraform `bundle plan -o json` will no longer include plan_version key, it's intended for direct engine only (([#4314](https://github.com/databricks/cli/pull/4314)))

### Dependency updates

### API Changes
