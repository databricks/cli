package patchwheel

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCalculateNewVersion(t *testing.T) {
	tests := []struct {
		name             string
		info             WheelInfo
		mtime            time.Time
		expectedVersion  string
		expectedFilename string
	}{
		{
			name: "basic version",
			info: WheelInfo{
				Distribution: "mypkg",
				Version:      "1.2.3",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:            time.Date(2025, 3, 4, 12, 34, 56, 780_123_321, time.UTC),
			expectedVersion:  "1.2.3+1741091696780123321",
			expectedFilename: "mypkg-1.2.3+1741091696780123321-py3-none-any.whl",
		},
		{
			name: "existing plus version",
			info: WheelInfo{
				Distribution: "mypkg",
				Version:      "1.2.3+local",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:            time.Date(2025, 3, 4, 12, 34, 56, 100_000_005, time.UTC),
			expectedVersion:  "1.2.3+1741091696100000005",
			expectedFilename: "mypkg-1.2.3+1741091696100000005-py3-none-any.whl",
		},
		{
			name: "complex distribution name",
			info: WheelInfo{
				Distribution: "my-pkg-name",
				Version:      "1.2.3",
				Tags:         []string{"py3", "none", "any"},
			},
			mtime:            time.Date(2025, 3, 4, 12, 34, 56, 0, time.UTC),
			expectedVersion:  "1.2.3+1741091696000000000",
			expectedFilename: "my-pkg-name-1.2.3+1741091696000000000-py3-none-any.whl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			newVersion, newFilename := calculateNewVersion(tt.info, tt.mtime)
			if newVersion != tt.expectedVersion {
				t.Errorf("expected version %s, got %s", tt.expectedVersion, newVersion)
			}
			if newFilename != tt.expectedFilename {
				t.Errorf("expected filename %s, got %s", tt.expectedFilename, newFilename)
			}
		})
	}
}

func TestParseWheelFilename(t *testing.T) {
	tests := []struct {
		filename         string
		wantDistribution string
		wantVersion      string
		wantTags         []string
		wantErr          bool
	}{
		{
			filename:         "myproj-0.1.0-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "./myproj-0.1.0-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "myproj-0.1.0+20240303123456-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0+20240303123456",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "my_proj_with_parts-0.1.0-py3-none-any.whl",
			wantDistribution: "my_proj_with_parts",
			wantVersion:      "0.1.0",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "subdir/myproj-0.1.0+20240303123456-py3-none-any.whl",
			wantDistribution: "myproj",
			wantVersion:      "0.1.0+20240303123456",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		// Test cases adapted from wheelodex/wheel-filename
		// https://github.com/wheelodex/wheel-filename/blob/f5a72b560d016cc9663c8b5a094c96dc338a2209/test/test_parse.py
		// MIT License: https://github.com/wheelodex/wheel-filename/blob/f5a72b560d016cc9663c8b5a094c96dc338a2209/LICENSE
		{
			filename:         "astrocats-0.3.2-universal-none-any.whl",
			wantDistribution: "astrocats",
			wantVersion:      "0.3.2",
			wantTags:         []string{"universal", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "bencoder.pyx-1.1.2-pp226-pp226-win32.whl",
			wantDistribution: "bencoder.pyx",
			wantVersion:      "1.1.2",
			wantTags:         []string{"pp226", "pp226", "win32"},
			wantErr:          false,
		},
		{
			filename:         "brotlipy-0.1.2-pp27-none-macosx_10_10_x86_64.whl",
			wantDistribution: "brotlipy",
			wantVersion:      "0.1.2",
			wantTags:         []string{"pp27", "none", "macosx_10_10_x86_64"},
			wantErr:          false,
		},
		{
			filename:         "brotlipy-0.3.0-pp226-pp226u-macosx_10_10_x86_64.whl",
			wantDistribution: "brotlipy",
			wantVersion:      "0.3.0",
			wantTags:         []string{"pp226", "pp226u", "macosx_10_10_x86_64"},
			wantErr:          false,
		},
		{
			filename:         "carbonara_archinfo-7.7.9.14.post1-py2-none-any.whl",
			wantDistribution: "carbonara_archinfo",
			wantVersion:      "7.7.9.14.post1",
			wantTags:         []string{"py2", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "coremltools-0.3.0-py2.7-none-any.whl",
			wantDistribution: "coremltools",
			wantVersion:      "0.3.0",
			wantTags:         []string{"py2.7", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "cvxopt-1.2.0-001-cp34-cp34m-macosx_10_6_intel.macosx_10_9_intel.macosx_10_9_x86_64.macosx_10_10_intel.macosx_10_10_x86_64.whl",
			wantDistribution: "cvxopt",
			wantVersion:      "1.2.0",
			wantTags:         []string{"001", "cp34", "cp34m", "macosx_10_6_intel.macosx_10_9_intel.macosx_10_9_x86_64.macosx_10_10_intel.macosx_10_10_x86_64"},
			wantErr:          false,
		},
		{
			filename:         "django_mbrowse-0.0.1-10-py2-none-any.whl",
			wantDistribution: "django_mbrowse",
			wantVersion:      "0.0.1",
			wantTags:         []string{"10", "py2", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "efilter-1!1.2-py2-none-any.whl",
			wantDistribution: "efilter",
			wantVersion:      "1!1.2",
			wantTags:         []string{"py2", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "line.sep-0.2.0.dev1-py2.py3-none-any.whl",
			wantDistribution: "line.sep",
			wantVersion:      "0.2.0.dev1",
			wantTags:         []string{"py2.py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "mayan_edms-1.1.0-1502100955-py2-none-any.whl",
			wantDistribution: "mayan_edms",
			wantVersion:      "1.1.0",
			wantTags:         []string{"1502100955", "py2", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "mxnet_model_server-1.0a5-20180816-py2.py3-none-any.whl",
			wantDistribution: "mxnet_model_server",
			wantVersion:      "1.0a5",
			wantTags:         []string{"20180816", "py2.py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "pip-18.0-py2.py3-none-any.whl",
			wantDistribution: "pip",
			wantVersion:      "18.0",
			wantTags:         []string{"py2.py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "polarTransform-2-1.0.0-py3-none-any.whl",
			wantDistribution: "polarTransform",
			wantVersion:      "2",
			wantTags:         []string{"1.0.0", "py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "psycopg2-2.7.5-cp37-cp37m-macosx_10_6_intel.macosx_10_9_intel.macosx_10_9_x86_64.macosx_10_10_intel.macosx_10_10_x86_64.whl",
			wantDistribution: "psycopg2",
			wantVersion:      "2.7.5",
			wantTags:         []string{"cp37", "cp37m", "macosx_10_6_intel.macosx_10_9_intel.macosx_10_9_x86_64.macosx_10_10_intel.macosx_10_10_x86_64"},
			wantErr:          false,
		},
		{
			filename:         "pyinterval-1.0.0-0-cp27-none-win32.whl",
			wantDistribution: "pyinterval",
			wantVersion:      "1.0.0",
			wantTags:         []string{"0", "cp27", "none", "win32"},
			wantErr:          false,
		},
		{
			filename:         "pypi_simple-0.1.0.dev1-py2.py3-none-any.whl",
			wantDistribution: "pypi_simple",
			wantVersion:      "0.1.0.dev1",
			wantTags:         []string{"py2.py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "PyQt3D-5.7.1-5.7.1-cp34.cp35.cp36-abi3-macosx_10_6_intel.whl",
			wantDistribution: "PyQt3D",
			wantVersion:      "5.7.1",
			wantTags:         []string{"5.7.1", "cp34.cp35.cp36", "abi3", "macosx_10_6_intel"},
			wantErr:          false,
		},
		{
			filename:         "qypi-0.4.1-py3-none-any.whl",
			wantDistribution: "qypi",
			wantVersion:      "0.4.1",
			wantTags:         []string{"py3", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "SimpleSteem-1.1.9-3.0-none-any.whl",
			wantDistribution: "SimpleSteem",
			wantVersion:      "1.1.9",
			wantTags:         []string{"3.0", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "simple_workflow-0.1.47-pypy-none-any.whl",
			wantDistribution: "simple_workflow",
			wantVersion:      "0.1.47",
			wantTags:         []string{"pypy", "none", "any"},
			wantErr:          false,
		},
		{
			filename:         "tables-3.4.2-3-cp27-cp27m-manylinux1_i686.whl",
			wantDistribution: "tables",
			wantVersion:      "3.4.2",
			wantTags:         []string{"3", "cp27", "cp27m", "manylinux1_i686"},
			wantErr:          false,
		},
		{
			filename:         "invalid-filename.txt",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
		{
			filename:         "not-enough-parts-py3.whl",
			wantDistribution: "",
			wantVersion:      "",
			wantTags:         nil,
			wantErr:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			info, err := ParseWheelFilename(tt.filename)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, tt.wantDistribution, info.Distribution, "distribution mismatch")
				require.Equal(t, tt.wantVersion, info.Version, "version mismatch")
				require.Equal(t, tt.wantTags, info.Tags, "tags mismatch")
			}
		})
	}
}

func TestParseError(t *testing.T) {
	tests := []string{
		"",
		"a-b",
		"a-b-c.whl",
		"a-b-c-d.whl",
		"a-b-c-d-e-f-g.whl",
	}
	for _, tt := range tests {
		t.Run(tt, func(t *testing.T) {
			info, err := ParseWheelFilename(tt)
			require.Error(t, err, "info=%v", info)
		})
	}
}
