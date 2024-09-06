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

	// Absolute paths.
	assert.False(t, IsLocalPath("/some/full/path"))
	assert.False(t, IsLocalPath("/Workspace/path/to/package"))
	assert.False(t, IsLocalPath("/Users/path/to/package"))

	// Paths with schemes.
	assert.False(t, IsLocalPath("dbfs://path/to/package"))
	assert.False(t, IsLocalPath("dbfs:/path/to/package"))
	assert.False(t, IsLocalPath("s3://path/to/package"))
	assert.False(t, IsLocalPath("abfss://path/to/package"))
}

func TestIsLibraryLocal(t *testing.T) {
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
		{path: "file://path/to/package/whl.whl", expected: true},
		{path: "", expected: false},
		{path: "pypipackage", expected: false},
		{path: "/Volumes/catalog/schema/volume/path.whl", expected: false},
		{path: "/Workspace/my_project/dist.whl", expected: false},
		{path: "-r /Workspace/my_project/requirements.txt", expected: false},
		{path: "s3://mybucket/path/to/package", expected: false},
		{path: "dbfs:/mnt/path/to/package", expected: false},
		{path: "beautifulsoup4", expected: false},
		{path: "beautifulsoup4==4.12.3", expected: false},
		{path: "beautifulsoup4 >= 4.12.3", expected: false},
		{path: "beautifulsoup4 < 4.12.3", expected: false},
		{path: "beautifulsoup4 ~= 4.12.3", expected: false},
		{path: "beautifulsoup4[security, tests]", expected: false},
		{path: "beautifulsoup4[security, tests] ~= 4.12.3", expected: false},
		{path: "beautifulsoup4>=1.0.0,<2.0.0", expected: false},
		{path: "beautifulsoup4>=1.0.0,~=1.2.0,<2.0.0", expected: false},
		{path: "https://github.com/pypa/pip/archive/22.0.2.zip", expected: false},
		{path: "pip @ https://github.com/pypa/pip/archive/22.0.2.zip", expected: false},
		{path: "requests [security] @ https://github.com/psf/requests/archive/refs/heads/main.zip", expected: false},
	}

	for i, tc := range testCases {
		require.Equalf(t, tc.expected, IsLibraryLocal(tc.path), "failed case: %d, path: %s", i, tc.path)
	}
}
