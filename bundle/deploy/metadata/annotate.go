package metadata

import (
	"context"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/bundle/config"
	"github.com/databricks/cli/libs/diag"
)

// annotations holds, per resource type, a function that creates the mutator that
// annotates resources of that type with the bundle deployment they belong to. They run
// in the canonical order of their types (the field order of [config.Resources]).
var annotations = config.NewResourceTypeRegistry[func() bundle.Mutator]("deployment annotation")

type annotateResources struct{}

// AnnotateResources returns a mutator that applies the registered annotation mutators in
// sequence. Like [bundle.ApplySeqContext], it stops at the first error.
func AnnotateResources() bundle.Mutator {
	return &annotateResources{}
}

func (m *annotateResources) Name() string {
	return "metadata.AnnotateResources"
}

func (m *annotateResources) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var mutators []bundle.Mutator
	for _, newMutator := range annotations.All() {
		mutators = append(mutators, newMutator())
	}
	bundle.ApplySeqContext(ctx, b, mutators...)
	return nil
}
