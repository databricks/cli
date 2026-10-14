package libraries

import (
	"github.com/databricks/cli/libs/sdktypes/jobs/v2"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsWorkspacePath(t *testing.T) {
	// Absolute paths with particular prefixes.
	assert.True(t, IsWorkspacePath("/Workspace/path/to/package"))
	assert.True(t, IsWorkspacePath("/Users/path/to/package"))
	assert.True(t, IsWorkspacePath("/Shared/path/to/package"))

	// Relative paths.
	assert.False(t, IsWorkspacePath("myfile.txt"))
	assert.False(t, IsWorkspacePath("./myfile.txt"))
	assert.False(t, IsWorkspacePath("../myfile.txt"))
}

func TestIsWorkspaceLibrary(t *testing.T) {
	// Workspace paths.
	assert.True(t, IsWorkspaceLibrary(&jobs.Library{Whl: new("/Workspace/path/to/file.whl")}))

	// Non-workspace paths.
	assert.False(t, IsWorkspaceLibrary(&jobs.Library{Whl: new("./file.whl")}))
	assert.False(t, IsWorkspaceLibrary(&jobs.Library{Jar: new("../target/some.jar")}))
	assert.False(t, IsWorkspaceLibrary(&jobs.Library{Jar: new("s3:/bucket/path/some.jar")}))

	// Empty.
	assert.False(t, IsWorkspaceLibrary(&jobs.Library{}))
}

func TestIsVolumesPath(t *testing.T) {
	// Absolute paths with particular prefixes.
	assert.True(t, IsVolumesPath("/Volumes/path/to/package"))

	// Relative paths.
	assert.False(t, IsVolumesPath("myfile.txt"))
	assert.False(t, IsVolumesPath("./myfile.txt"))
	assert.False(t, IsVolumesPath("../myfile.txt"))
}
