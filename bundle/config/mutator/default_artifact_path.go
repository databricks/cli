package mutator

import (
	"context"
	"fmt"
	"path"

	"github.com/databricks/bricks/bundle"
)

type defaultArtifactPath struct{}

// DefaultArtifactPath configures the artifact path if it isn't already set.
func DefaultArtifactPath() bundle.Mutator {
	return &defaultArtifactPath{}
}

func (m *defaultArtifactPath) Name() string {
	return "DefaultArtifactPath"
}

func (m *defaultArtifactPath) Apply(ctx context.Context, b *bundle.Bundle) ([]bundle.Mutator, error) {
	if b.Config.Workspace.ArtifactPath.IsSet() {
		return nil, nil
	}

	me := b.Config.Workspace.CurrentUser
	if me == nil {
		return nil, fmt.Errorf("cannot configured default artifact path if current user isn't set")
	}

	// We assume we deal with notebooks only for the time being.
	// When we need to upload wheel files or other non-notebook files,
	// the workspace must support "Files in Workspace".
	// If it doesn't, we need to resort to storing artifacts on DBFS.
	home := fmt.Sprintf("/Users/%s", me.UserName)
	root := path.Join(home, ".bundle", b.Config.Bundle.Name, b.Config.Bundle.Environment, "artifacts")
	b.Config.Workspace.ArtifactPath.Workspace = root
	return nil, nil
}
