// Package nextchanges exposes the next release version to the build.
//
// It lives in this directory because go:embed cannot reach a parent directory:
// embedding the version file from internal/build would require a second copy of
// the value, which could then drift from this one.
package nextchanges

import _ "embed"

// Version is the next release version, e.g. "1.12.0\n". Callers must trim it.
// The release tooling bumps the embedded file after each release; see README.md.
//
//go:embed version
var Version string
