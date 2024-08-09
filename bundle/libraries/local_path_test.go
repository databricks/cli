package libraries

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLocalPath(t *testing.T) {
	// Relative paths, paths with the file scheme, and Windows paths.
	assert.True(t, IsLocalPath("some/local/path"))
	assert.True(t, IsLocalPath("./some/local/path"))
	assert.True(t, IsLocalPath("file://path/to/package"))
	assert.True(t, IsLocalPath("C:\\path\\to\\package"))

	assert.True(t, IsLocalPath("myfile.txt"))
	assert.True(t, IsLocalPath("./myfile.txt"))
	assert.True(t, IsLocalPath("../myfile.txt"))
	assert.True(t, IsLocalPath("file:///foo/bar/myfile.txt"))

	// Remote paths.
	assert.False(t, IsLocalPath("/some/full/path"))
	assert.False(t, IsLocalPath("/Workspace/path/to/package"))
	assert.False(t, IsLocalPath("/Users/path/to/package"))

	// Paths with schemes.
	assert.False(t, IsLocalPath("dbfs://path/to/package"))
	assert.False(t, IsLocalPath("dbfs:/path/to/package"))
	assert.False(t, IsLocalPath("s3://path/to/package"))
	assert.False(t, IsLocalPath("abfss://path/to/package"))
}

func TestIsRemotePath(t *testing.T) {
	// Paths with schemes.
	assert.True(t, IsRemotePath("dbfs://path/to/package"))
	assert.True(t, IsRemotePath("dbfs:/path/to/package"))
	assert.True(t, IsRemotePath("s3://path/to/package"))
	assert.True(t, IsRemotePath("abfss://path/to/package"))

	// Remote paths.
	assert.True(t, IsRemotePath("/Workspace/path/to/package"))
	assert.True(t, IsRemotePath("/Users/path/to/package"))

	// Relative paths, paths with the file scheme, and Windows paths.
	assert.False(t, IsRemotePath("some/local/path"))
	assert.False(t, IsRemotePath("./some/local/path"))
	assert.False(t, IsRemotePath("file://path/to/package"))

	assert.False(t, IsRemotePath("myfile.txt"))
	assert.False(t, IsRemotePath("./myfile.txt"))
	assert.False(t, IsRemotePath("../myfile.txt"))

	// Local absolute paths.
	assert.False(t, IsRemotePath("/some/full/path"))
	assert.False(t, IsRemotePath("file:///foo/bar/myfile.txt"))
	assert.False(t, IsRemotePath("C:\\path\\to\\package"))

}

func TestIsEnvironmentDependencyLocal(t *testing.T) {
	testCases := [](struct {
		path     string
		expected bool
	}){
		{path: "local/*.whl", expected: true},
		{path: "local/test.whl", expected: true},
		{path: "./local/*.whl", expected: true},
		{path: ".\\local\\*.whl", expected: true},
		{path: "./local/mypath.whl", expected: true},
		{path: ".\\local\\mypath.whl", expected: true},
		{path: "../local/*.whl", expected: true},
		{path: "..\\local\\*.whl", expected: true},
		{path: "./../local/*.whl", expected: true},
		{path: ".\\..\\local\\*.whl", expected: true},
		{path: "../../local/*.whl", expected: true},
		{path: "..\\..\\local\\*.whl", expected: true},
		{path: "pypipackage", expected: false},
		{path: "/Volumes/catalog/schema/volume/path.whl", expected: false},
		{path: "/Workspace/my_project/dist.whl", expected: false},
		{path: "-r /Workspace/my_project/requirements.txt", expected: false},
	}

	for i, tc := range testCases {
		require.Equalf(t, tc.expected, IsLibraryLocal(tc.path), "failed case: %d, path: %s", i, tc.path)
	}
}
