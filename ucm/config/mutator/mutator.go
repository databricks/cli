package mutator

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/loader"
	"github.com/databricks/cli/ucm/config/validate"
)

func DefaultMutators(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		loader.ProcessRootIncludes(),
		FlattenNestedResources(),
		InheritCatalogTags(),
		DefineDefaultTarget(),

		// Mirrors bundle/config/mutator/mutator.go: catch duplicate resource
		// keys on the raw post-load config before target overrides merge.
		// Also runs again as part of validate.All during the validate phase.
		validate.UniqueResourceKeys(),
	)
}
