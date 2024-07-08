package bundle

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/diag"
)

// Control signal error that can be used to break out of a sequence of mutators.
var ErrorBreakSequence = errors.New("break sequence")
var DiagnosticBreakSequence = diag.FromErr(ErrorBreakSequence)

type seqMutator struct {
	mutators []Mutator
}

func (s *seqMutator) Name() string {
	return "seq"
}

func (s *seqMutator) Apply(ctx context.Context, b *Bundle) diag.Diagnostics {
	var diags diag.Diagnostics
	for _, m := range s.mutators {
		nd := Apply(ctx, b, m)

		// Break out of the sequence. Filter the ErrorBreakSequence error so that
		// it does not show up to the user.
		if nd.ContainsError(ErrorBreakSequence) {
			diags.Extend(nd.FilterError(ErrorBreakSequence))
			break
		}

		if diags.HasError() {
			diags.Extend(nd)
			break
		}
	}
	return diags
}

func Seq(ms ...Mutator) Mutator {
	return &seqMutator{mutators: ms}
}

