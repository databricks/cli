#!/bin/bash
set -e
# With golangci-lint, if there are any compliation issues, they formatters autofix won't be applied.
# https://github.com/golangci/golangci-lint/issues/5257
# However, running goimports first alone will actually fix some of the compilation issues.
# For this reason, this script runs golangci-lint in two stages:
golangci-lint run --fix --no-config --disable-all --enable gofumpt,goimports $@
exec golangci-lint run --fix $@
