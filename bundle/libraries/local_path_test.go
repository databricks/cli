package libraries

import (
	"testing"

	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLocalPath(t *testing.T) {
	// Relative paths, paths with the file scheme, and Windows paths.
	assert.True(t, IsLocalPath("some/local/path"))
	assert.True(t, IsLocalPath("./some/local/path"))
	assert.True(t, IsLocalPath("file://path/to/package"))
	assert.True(t, IsLocalPath("C:\\path\\to\\package"))
	assert.True(t, IsLocalPath("/some/full/path"))
	assert.True(t, IsLocalPath("myfile.txt"))
	assert.True(t, IsLocalPath("./myfile.txt"))
	assert.True(t, IsLocalPath("../myfile.txt"))
	assert.True(t, IsLocalPath("file:///foo/bar/myfile.txt"))

	// Remote paths.
	assert.False(t, IsLocalPath("/Workspace/path/to/package"))
	assert.False(t, IsLocalPath("/Users/path/to/package"))

	// Paths with schemes.
	assert.False(t, IsLocalPath("dbfs://path/to/package"))
	assert.False(t, IsLocalPath("dbfs:/path/to/package"))
	assert.False(t, IsLocalPath("s3://path/to/package"))
	assert.False(t, IsLocalPath("abfss://path/to/package"))
}

func TestIsLocalLibrary(t *testing.T) {
	// Local paths.
	assert.True(t, IsLocalLibrary(&compute.Library{Whl: "./file.whl"}))
	assert.True(t, IsLocalLibrary(&compute.Library{Jar: "../target/some.jar"}))

	// Non-local paths.
	assert.False(t, IsLocalLibrary(&compute.Library{Whl: "/Workspace/path/to/file.whl"}))
	assert.False(t, IsLocalLibrary(&compute.Library{Jar: "s3:/bucket/path/some.jar"}))

	// Empty.
	assert.False(t, IsLocalLibrary(&compute.Library{}))
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
