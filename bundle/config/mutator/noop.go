package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
)

type noop struct{}

func (*noop) Apply(context.Context, *bundle.Bundle) diag.Diagnostics {
	return nil
}

func (*noop) Name() string {
	return "NoOp"
}

func NoOp() bundle.Mutator {
	return &noop{}
}
