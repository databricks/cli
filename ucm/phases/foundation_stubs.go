package phases

import (
	"context"

	"github.com/databricks/cli/ucm"
)

// LibLocationMap mirrors bundle/phases.LibLocationMap. Maps a library
// reference (path or coordinate) to its uploaded location.
//
// UCM resources are pure Unity Catalog metadata (catalogs, schemas, volumes,
// external locations, storage credentials, connections, grants, tag
// validation rules) — none of which carry uploaded artifacts. The type is
// retained as a permanent no-op shape-aligner for the cmd/ucm/utils.ProcessUcm
// fork (#98), not as a placeholder for future implementation.
type LibLocationMap map[string]string

// BuildArtifacts is a permanent no-op. Bundle's phases.Build walks
// jobs/pipelines/etc. for libraries and uploads them; UCM has no
// artifact-bearing resource type and doesn't intend to gain one — UC
// objects are metadata. The function is retained so cmd/ucm/utils.ProcessUcm
// can call it under opts.Build without divergence from bundle's flow shape.
//
// Named BuildArtifacts (not Build) to avoid colliding with the existing
// terraform-render phase Build(ctx, u, opts).
func BuildArtifacts(_ context.Context, _ *ucm.Ucm) LibLocationMap {
	return nil
}
