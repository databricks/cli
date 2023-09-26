package mutator

import (
	"context"

	"github.com/databricks/cli/bundle"
)

type noop struct{}

func (*noop) Apply(context.Context, *bundle.Bundle) error {
	return nil
}

func (*noop) Name() string {
	return "NoOp"
}

func NoOp() bundle.Mutator {
	return &noop{}
}
