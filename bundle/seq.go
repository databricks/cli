package bundle

import (
	"context"
	"errors"

	"github.com/databricks/cli/libs/diag"
)

// Control signal error that can be returned by a mutator to break out of a sequence.
var ErrorSequenceBreak = errors.New("break sequence")

// Convenient diagnostic that wraps ErrorSequenceBreak.
var DiagnosticSequenceBreak = diag.FromErr(ErrorSequenceBreak)

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

		// Remove the ErrorSequenceBreak error from the diagnostics. It's a control
		// signal and should not be shown to the user.
		if nd.ContainsError(ErrorSequenceBreak) {
			nd.RemoveError(ErrorSequenceBreak)
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
