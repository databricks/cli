package configsync

import "github.com/databricks/cli/bundle/deployplan"

// FileChange represents a change to a bundle configuration file
type FileChange struct {
	Path            string `json:"path"`
	OriginalContent string `json:"originalContent"`
	ModifiedContent string `json:"modifiedContent"`
}

// DiffOutput represents the complete output of the config-remote-sync command
type DiffOutput struct {
	Files   []FileChange                  `json:"files"`
	Changes map[string]deployplan.Changes `json:"changes"`
}
