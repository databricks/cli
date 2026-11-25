package resourcemutator

import (
	"context"
	"fmt"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config/stages"
	"github.com/databricks/cli/libs/diag"
)

// The runner lives here rather than in package stages because this package registers the
// stage mutators: every caller of ApplyStage links them, so a stage never runs empty.
type applyStage struct {
	stage stages.Stage
}

// ApplyStage returns a mutator that applies the mutators registered for stage in sequence.
// Like [bundle.ApplySeqContext], it stops at the first error.
func ApplyStage(stage stages.Stage) bundle.Mutator {
	return &applyStage{stage: stage}
}

func (m *applyStage) Name() string {
	return fmt.Sprintf("ApplyStage(%s)", m.stage)
}

func (m *applyStage) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	bundle.ApplySeqContext(ctx, b, stages.Mutators(m.stage)...)
	return nil
}
