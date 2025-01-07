#!/bin/bash
set -uo pipefail
# With golangci-lint, if there are any compliation issues, then formatters' autofix won't be applied.
# https://github.com/golangci/golangci-lint/issues/5257

golangci-lint run --fix "$@"
lint_exit_code=$?

if [ $lint_exit_code -ne 0 ]; then
    # These linters work in presence of compilation issues when run alone, so let's get these fixes at least.
    golangci-lint run --enable-only="gofmt,gofumpt,goimports" --fix "$@"
fi

exit $lint_exit_code
