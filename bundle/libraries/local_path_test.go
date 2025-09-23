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
		{path: "-r ../requirements.txt", expected: false},
		{path: "-r /Workspace/my_project/requirements.txt", expected: false},
		{path: "s3://mybucket/path/to/package", expected: false},
		{path: "dbfs:/mnt/path/to/package", expected: false},
		{path: "beautifulsoup4", expected: false},

		// Check the possible version specifiers as in PEP 440
		// https://peps.python.org/pep-0440/#public-version-identifiers
		{path: "beautifulsoup4==4", expected: false},
		{path: "beautifulsoup4==4.12", expected: false},
		{path: "beautifulsoup4==4.12.3", expected: false},
		{path: "beautifulsoup4==4.12.3.dev1", expected: false},
		{path: "beautifulsoup4==4.12.3.a1", expected: false},
		{path: "beautifulsoup4==4.12.3.rc2", expected: false},
		{path: "beautifulsoup4==4.12.3.rc2.dev1", expected: false},
		{path: "beautifulsoup4==4.12.3+abc.5", expected: false},
		{path: "beautifulsoup4==1!4.12.3", expected: false},

		{path: "beautifulsoup4 >= 4.12.3", expected: false},
		{path: "beautifulsoup4 < 4.12.3", expected: false},
		{path: "beautifulsoup4 ~= 4.12.3", expected: false},
		{path: "beautifulsoup4[security, tests]", expected: false},
		{path: "beautifulsoup4[security, tests] ~= 4.12.3", expected: false},
		{path: "beautifulsoup4>=1.0.0,<2.0.0", expected: false},
		{path: "beautifulsoup4>=1.0.0,~=1.2.0,<2.0.0", expected: false},
		{path: "beautifulsoup4>=1.0.0+abc.5,~=1.2.0.rc2.dev1,<2.0.0.a1", expected: false},
		{path: "https://github.com/pypa/pip/archive/22.0.2.zip", expected: false},
		{path: "pip @ https://github.com/pypa/pip/archive/22.0.2.zip", expected: false},
		{path: "requests [security] @ https://github.com/psf/requests/archive/refs/heads/main.zip", expected: false},

		// Test pip options: these should not be treated as local paths
		{path: "-e ..", expected: false},
		{path: "-e ../myproject", expected: false},
		{path: "-e ./local/package", expected: false},
		{path: "-r requirements.txt", expected: false},
		{path: "-f /path/to/wheels", expected: false},
		{path: "--find-links /path/to/wheels", expected: false},
		{path: "--index-url https://pypi.org/simple", expected: false},
	}

	for i, tc := range testCases {
		require.Equalf(t, tc.expected, IsLibraryLocal(tc.path), "failed case: %d, path: %s", i, tc.path)
	}
}

func TestIsLocalRequirementsFile(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		isLocal  bool
	}{
		{
			name:     "valid requirements file with space",
			input:    "-r requirements.txt",
			expected: "requirements.txt",
			isLocal:  true,
		},
		{
			name:     "remote requirements file",
			input:    "-r /Workspace/Users/requirements.txt",
			expected: "/Workspace/Users/requirements.txt",
			isLocal:  false,
		},
		{
			name:     "not a requirements file",
			input:    "some.txt",
			expected: "",
			isLocal:  false,
		},
		{
			name:     "-r with no space",
			input:    "-rrequirements.txt",
			expected: "",
			isLocal:  false,
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
			isLocal:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, isLocal := IsLocalRequirementsFile(tt.input)
			require.Equal(t, tt.expected, got)
			require.Equal(t, tt.isLocal, isLocal)
		})
	}
}

func TestContainsPipFlag(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		// Single-dash options
		{"-e ..", true},
		{"-e ../myproject", true},
		{"-e ./local/package", true},
		{"-r requirements.txt", true},
		{"-f /path/to/wheels", true},
		{"-i https://pypi.org/simple", true},

		// Double-dash options
		{"--find-links /path/to/wheels", true},
		{"--index-url https://pypi.org/simple", true},
		{"--extra-index-url https://pypi.org/simple", true},
		{"--trusted-host pypi.org", true},
		{"--editable ../myproject", true},

		// Not pip options
		{"beautifulsoup4", false},
		{"../local/package", false},
		{"./local/package", false},
		{"requirements.txt", false},
		{"", false},
		{"-", false},
		{"--", false},
		{"-e", false}, // No space after -e
		{"-r", false}, // No space after -r
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			got := ContainsPipFlag(tc.input)
			assert.Equal(t, tc.expected, got, "input: %s", tc.input)
		})
	}
}
