package libraries

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
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
	assert.True(t, IsWorkspaceLibrary(&compute.Library{Whl: "/Workspace/path/to/file.whl"}))

	// Non-workspace paths.
	assert.False(t, IsWorkspaceLibrary(&compute.Library{Whl: "./file.whl"}))
	assert.False(t, IsWorkspaceLibrary(&compute.Library{Jar: "../target/some.jar"}))
	assert.False(t, IsWorkspaceLibrary(&compute.Library{Jar: "s3:/bucket/path/some.jar"}))

	// Empty.
	assert.False(t, IsWorkspaceLibrary(&compute.Library{}))
}

func TestIsVolumesPath(t *testing.T) {
	// Absolute paths with particular prefixes.
	assert.True(t, IsVolumesPath("/Volumes/path/to/package"))

	// Relative paths.
	assert.False(t, IsVolumesPath("myfile.txt"))
	assert.False(t, IsVolumesPath("./myfile.txt"))
	assert.False(t, IsVolumesPath("../myfile.txt"))
}
