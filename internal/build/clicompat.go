package build

import _ "embed"

// AgentSkillsFallbackVersion is used by direct go builds whose embedded manifest
// still contains "latest". Release and task builds resolve it before embedding.
const AgentSkillsFallbackVersion = "0.2.30"

// CLICompatManifestJSON is the cli-compat.json manifest embedded at compile time.
// Release and task builds replace skills "latest" with a concrete release.
// Used as the last-resort fallback when both remote fetch and local cache fail.
//
//go:embed cli-compat.json
var CLICompatManifestJSON []byte
