// Package nextchanges exposes the next release version to the build.
//
// It lives in this directory because go:embed cannot reach a parent directory:
// embedding the version file from internal/build would require a second copy of
// the value, which could then drift from this one.
package nextchanges

import (
	_ "embed"
	"strings"
)

// versionFile is the raw contents of the version file. It has a trailing newline
// (the whitespace linter requires one), so it is not exported directly.
//
//go:embed version
var versionFile string

// Version is the next release version, e.g. "1.12.0". The release tooling bumps
// the embedded file after each release; see README.md.
var Version = strings.TrimSpace(versionFile)
