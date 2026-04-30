#!/usr/bin/env pwsh
$ErrorActionPreference = 'Stop'
& go tool -modfile="$PSScriptRoot/tools/task/go.mod" task @args
exit $LASTEXITCODE
