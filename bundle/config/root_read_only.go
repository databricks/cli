package config

import (
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type ReadOnlyConfig struct {
	r Root
}

func ReadOnly(r Root) ReadOnlyConfig {
	return ReadOnlyConfig{r: r}
}

func (r ReadOnlyConfig) Artifacts() Artifacts {
	return r.r.Artifacts
}

func (r ReadOnlyConfig) Bundle() Bundle {
	return r.r.Bundle
}

func (r ReadOnlyConfig) Environments() map[string]*Target {
	return r.r.Environments
}

func (r ReadOnlyConfig) Experimental() *Experimental {
	return r.r.Experimental
}

func (r ReadOnlyConfig) Include() []string {
	return r.r.Include
}

func (r ReadOnlyConfig) Permissions() []resources.Permission {
	return r.r.Permissions
}

func (r ReadOnlyConfig) Resources() Resources {
	return r.r.Resources
}

func (r ReadOnlyConfig) Targets() map[string]*Target {
	return r.r.Targets
}

func (r ReadOnlyConfig) RunAs() *jobs.JobRunAs {
	return r.r.RunAs
}

func (r ReadOnlyConfig) Sync() Sync {
	return r.r.Sync
}

func (r ReadOnlyConfig) Variables() map[string]*variable.Variable {
	return r.r.Variables
}

func (r ReadOnlyConfig) Workspace() Workspace {
	return r.r.Workspace
}

func (r ReadOnlyConfig) GetLocation(path string) dyn.Location {
	return r.r.GetLocation(path)
}
