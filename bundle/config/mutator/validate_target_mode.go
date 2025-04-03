package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

type validateTargetMode struct{}

// ValidateTargetMode validates that bundle resources have an adequate configuration
// for a selected target mode.
func ValidateTargetMode() bundle.Mutator {
	return &validateTargetMode{}
}

func (v validateTargetMode) Name() string {
	return "ValidateTargetMode"
}

func (v validateTargetMode) Apply(_ context.Context, b *bundle.Bundle) diag.Diagnostics {
	if b.Config.Bundle.Mode == config.Production {
		return validateProductionPipelines(b)
	} else {
		return nil
	}
}

func validateProductionPipelines(b *bundle.Bundle) diag.Diagnostics {
	r := b.Config.Resources
	for i := range r.Pipelines {
		if r.Pipelines[i].Development {
			return diag.Errorf("target with 'mode: production' cannot include a pipeline with 'development: true'")
		}
	}

	return nil
}
