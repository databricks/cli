package config

import (
	"github.com/databricks/databricks-sdk-go/databricks"
	"github.com/databricks/databricks-sdk-go/workspaces"
)

// Workspace defines configurables at the workspace level.
type Workspace struct {
	// TODO: Add all unified authentication configurables.
	Host    string `json:"host,omitempty"`
	Profile string `json:"profile,omitempty"`
}

func (w *Workspace) Client() *workspaces.WorkspacesClient {
	config := databricks.Config{
		Host:    w.Host,
		Profile: w.Profile,
	}

	return workspaces.New(&config)
}
