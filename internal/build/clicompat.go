package build

import _ "embed"

// AgentSkillsFallbackVersion is the tested release used when fetching the latest
// skills release fails. Updating it requires a new CLI build.
const AgentSkillsFallbackVersion = "0.2.30"

// CLICompatManifestJSON is the cli-compat.json manifest embedded at compile time.
// Used as the last-resort fallback when both remote fetch and local cache fail.
//
//go:embed cli-compat.json
var CLICompatManifestJSON []byte
