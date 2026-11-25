package resources

import (
	"github.com/databricks/cli/libs/diag"
)

// ProductionModeValidator is implemented by resource types with settings that
// a target with 'mode: production' rejects.
type ProductionModeValidator interface {
	ValidateProductionMode() diag.Diagnostics
}

// ExplicitRunAsReporter is implemented by resource types that 'mode: production'
// inspects to tell whether every resource sets the principal it runs as.
type ExplicitRunAsReporter interface {
	IsRunAsSet() bool
}
