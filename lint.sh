#!/bin/bash
set -euo pipefail
# With golangci-lint, if there are any compliation issues, then formatters' autofix won't be applied.
# https://github.com/golangci/golangci-lint/issues/5257
# However, running goimports first alone will actually fix some of the compilation issues.
# Fixing formatting is also reasonable thing to do.
# For this reason, this script runs golangci-lint in two stages:
golangci-lint run --fix --no-config --disable-all --enable gofumpt,goimports $@
exec golangci-lint run --fix $@
