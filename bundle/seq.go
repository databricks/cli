package bundle

import (
	"context"

	"github.com/databricks/cli/libs/diag"
)

type seqMutator struct {
	mutators []Mutator
}

func (s *seqMutator) Name() string {
	return "seq"
}

func (s *seqMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, m := range s.mutators {
		diags = diags.Extend(Apply(ctx, b, m))
		if diags.HasError() {
			break
		}
	}
	return diags
}

func Seq(ms ...Mutator) Mutator {
	return &seqMutator{mutators: ms}
}
