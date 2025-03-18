# NEXT CHANGELOG

## Release v0.243.1

### CLI
* Fixed "can't evaluate field Name in type interface{}" for "databricks queries list" command ([#2451](https://github.com/databricks/cli/pull/2451))
* Fixed `query-history list` command failing with 'executing "command" at <.>: range cant iterate over' ([#2506](https://github.com/databricks/cli/pull/2506))

### Bundles

### Internal

### API Changes
* Added `databricks genie execute-message-attachment-query` command.
* Added `databricks genie get-message-attachment-query-result` command.
* `databricks genie execute-message-query` marked as Deprecated.
* `databricks genie get-message-query-result-by-attachment` marked as Deprecated.
