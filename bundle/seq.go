package bundle

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/diag"
)

// Control signal error that can be used to break out of a sequence of mutators.
// TODO: Are better names possible?
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
		hasError := nd.HasError()

		// Remove the ErrorBreakSequence error from the diagnostics. It's a control
		// signal and should not be shown to the user.
		if nd.ContainsError(ErrorBreakSequence) {
			nd.RemoveError(ErrorBreakSequence)
		}

		// Extend the diagnostics with the diagnostics from the current mutator.
		diags = diags.Extend(nd)

		// Break out of the sequence if there is an error.
		if hasError {
			break
		}
	}
	return diags
}

func Seq(ms ...Mutator) Mutator {
	return &seqMutator{mutators: ms}
}
