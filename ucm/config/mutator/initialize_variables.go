package mutator

import (
	"context"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/variable"
)

type initializeVariables struct{}

// InitializeVariables replaces any nil entries in Config.Variables with a
// freshly-allocated zero value. Shorthand variable definitions (e.g. just a
// key with no body) land as nil after YAML parsing and would otherwise panic
// later mutators.
func InitializeVariables() ucm.Mutator {
	return &initializeVariables{}
}

func (m *initializeVariables) Name() string { return "InitializeVariables" }

func (m *initializeVariables) Apply(_ context.Context, u *ucm.Ucm) diag.Diagnostics {
	for k, v := range u.Config.Variables {
		if v == nil {
			u.Config.Variables[k] = &variable.Variable{}
		}
	}
	return nil
}
