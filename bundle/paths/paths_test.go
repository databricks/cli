package paths

import (
	"testing"

	"github.com/databricks/cli/bundle/config"
	"github.com/stretchr/testify/assert"
)

func TestWorkspacePathsGoverning(t *testing.T) {
	// All logical paths nested under root_path collapse to a single prefix, and the
	// state path is governed by (inherits the ACL of) that root prefix.
	nested := CollectUniqueWorkspacePathPrefixes(config.Workspace{
		RootPath:     "/Workspace/u/bundle",
		ArtifactPath: "/Workspace/u/bundle/artifacts",
		FilePath:     "/Workspace/u/bundle/files",
		StatePath:    "/Workspace/u/bundle/state",
		ResourcePath: "/Workspace/u/bundle/resources",
	})
	assert.Equal(t, []string{"/Workspace/u/bundle"}, nested.Paths)
	assert.Equal(t, "/Workspace/u/bundle", nested.Governing("/Workspace/u/bundle/state"))

	// A state path outside root_path is its own prefix and governs itself.
	separate := CollectUniqueWorkspacePathPrefixes(config.Workspace{
		RootPath:     "/Workspace/u/bundle",
		ArtifactPath: "/Workspace/u/bundle/artifacts",
		FilePath:     "/Workspace/u/bundle/files",
		StatePath:    "/Workspace/Shared/state",
		ResourcePath: "/Workspace/u/bundle/resources",
	})
	assert.Equal(t, []string{"/Workspace/u/bundle", "/Workspace/Shared/state"}, separate.Paths)
	assert.Equal(t, "/Workspace/Shared/state", separate.Governing("/Workspace/Shared/state"))

	// A path governed by no prefix (a sibling) returns an empty string.
	assert.Empty(t, nested.Governing("/Workspace/u/bundle-2/state"))
}
