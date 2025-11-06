# NEXT CHANGELOG

## Release v0.277.0

### Notable Changes

### CLI

### Dependency updates

### Bundles
* Add validation that served_models and served_entities are not used at the same time. Add client side translation logic. ([#3880](https://github.com/databricks/cli/pull/3880))
* Gracefully handle interrupts (SIGINT, SIGTERM, SIGHUP, SIGQUIT) during bundle deployment and destruction by releasing locks before exiting ([#3758](https://github.com/databricks/cli/pull/3758))

### API Changes
