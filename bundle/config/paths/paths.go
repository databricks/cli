package paths

import (
	"github.com/databricks/cli/libs/dyn"
)

type Paths struct {
	// Absolute path on the local file system to the configuration file that holds
	// the definition of this resource.
	ConfigFilePath string `json:"-" bundle:"readonly"`

	// DynamicValue stores the [dyn.Value] of the containing struct.
	// This assumes that this struct is always embedded.
	DynamicValue dyn.Value `json:"-"`
}

func (p *Paths) ConfigureConfigFilePath() {
	if !p.DynamicValue.IsValid() {
		panic("DynamicValue not set")
	}
	p.ConfigFilePath = p.DynamicValue.Location().File
}
