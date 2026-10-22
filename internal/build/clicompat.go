package build

import _ "embed"

// CLICompatManifestJSON is the cli-compat.json manifest embedded at compile time.
// Release and task builds replace skills "latest" with a concrete release.
// Used as the last-resort fallback when both remote fetch and local cache fail.
//
//go:embed cli-compat.json
var CLICompatManifestJSON []byte
