# NEXT CHANGELOG

## Release v1.6.0

### Notable Changes

### CLI

### Bundles
* Fixed persistent drift on `model_serving_endpoints` in the direct engine, where a clean deploy was always followed by a spurious update because the backend-defaulted `traffic_config` was treated as a change. A suppressed field also no longer triggers an unnecessary config update on otherwise-unrelated changes.

### Dependency updates

### API Changes
