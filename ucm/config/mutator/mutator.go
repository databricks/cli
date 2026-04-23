package mutator

import (
	"context"

	"github.com/databricks/cli/ucm"
	"github.com/databricks/cli/ucm/config/loader"
)

func DefaultMutators(ctx context.Context, u *ucm.Ucm) {
	ucm.ApplySeqContext(ctx, u,
		loader.ProcessRootIncludes(),
		FlattenNestedResources(),
		InheritCatalogTags(),
		DefineDefaultTarget(),
	)
}
