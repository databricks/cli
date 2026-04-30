package build

import (
	_ "embed"
	"encoding/json"
	"sync"
)

//go:embed dep_versions.json
var depVersionsJSON []byte

// DepVersions holds build-time resolved dependency versions.
type DepVersions struct {
	AppKit      string `json:"appkit"`
	AgentSkills string `json:"skills"`
}

var depVersions = sync.OnceValue(func() DepVersions {
	var dv DepVersions
	_ = json.Unmarshal(depVersionsJSON, &dv)
	return dv
})

// GetDepVersions returns the build-time resolved dependency versions.
// Returns zero-value DepVersions if not set (dev builds).
func GetDepVersions() DepVersions {
	return depVersions()
}
