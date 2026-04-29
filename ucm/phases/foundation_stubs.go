package phases

import (
	"context"

	"github.com/databricks/cli/ucm"
)

// LibLocationMap mirrors bundle/phases.LibLocationMap. Maps a library
// reference (path or coordinate) to its uploaded location.
//
// UCM does not yet have an artifact/library concept, so the type ships as a
// placeholder used only to keep the cmd/ucm/utils.ProcessUcm fork (#98)
// shape-aligned with bundle. Tracked in #101.
type LibLocationMap map[string]string

// BuildArtifacts is a no-op stub for the bundle.phases.Build artifact-walk
// step. Bundle's Build returns a LibLocationMap describing every uploaded
// library; UCM has no artifact tree today and returns nil.
//
// Tracked in #101 — replace with a real implementation when ucm gains
// artifacts. Named BuildArtifacts (not Build) to avoid colliding with the
// existing terraform-render phase Build(ctx, u, opts).
func BuildArtifacts(_ context.Context, _ *ucm.Ucm) LibLocationMap {
	return nil
}
