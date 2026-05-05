package build

import _ "embed"

// EmbeddedManifestJSON is the cli-compat.json manifest embedded at compile time.
// Used as the last-resort fallback when both remote fetch and local cache fail.
//
//go:embed cli-compat.json
var EmbeddedManifestJSON []byte
