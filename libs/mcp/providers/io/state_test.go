package io

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProjectState(t *testing.T) {
	state := NewProjectState()
	assert.Equal(t, StateScaffolded, state.State)
	assert.Nil(t, state.Data)
}

func TestProjectState_Validate(t *testing.T) {
	scaffolded := NewProjectState()
	checksum := "abc123def456"

	validated := scaffolded.Validate(checksum)

	assert.Equal(t, StateValidated, validated.State)
	require.NotNil(t, validated.Data)

	data, ok := validated.Data.(ValidatedData)
	require.True(t, ok)
	assert.Equal(t, checksum, data.Checksum)
	assert.WithinDuration(t, time.Now().UTC(), data.ValidatedAt, time.Second)
}

func TestProjectState_Deploy(t *testing.T) {
	tests := []struct {
		name        string
		state       *ProjectState
		expectError bool
		errorMsg    string
	}{
		{
			name:        "cannot deploy from scaffolded",
			state:       NewProjectState(),
			expectError: true,
			errorMsg:    "cannot deploy: project not validated",
		},
		{
			name: "can deploy from validated",
			state: &ProjectState{
				State: StateValidated,
				Data: ValidatedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "abc123",
				},
			},
			expectError: false,
		},
		{
			name: "cannot deploy from already deployed",
			state: &ProjectState{
				State: StateDeployed,
				Data: DeployedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "abc123",
					DeployedAt:  time.Now().UTC(),
				},
			},
			expectError: true,
			errorMsg:    "cannot deploy: project already deployed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			deployed, err := tt.state.Deploy()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, deployed)
			} else {
				require.NoError(t, err)
				require.NotNil(t, deployed)
				assert.Equal(t, StateDeployed, deployed.State)

				data, ok := deployed.Data.(DeployedData)
				require.True(t, ok)
				assert.WithinDuration(t, time.Now().UTC(), data.DeployedAt, time.Second)
			}
		})
	}
}

func TestProjectState_Checksum(t *testing.T) {
	tests := []struct {
		name           string
		state          *ProjectState
		expectedSum    string
		expectedExists bool
	}{
		{
			name:           "scaffolded has no checksum",
			state:          NewProjectState(),
			expectedSum:    "",
			expectedExists: false,
		},
		{
			name: "validated has checksum",
			state: &ProjectState{
				State: StateValidated,
				Data: ValidatedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "validated_checksum",
				},
			},
			expectedSum:    "validated_checksum",
			expectedExists: true,
		},
		{
			name: "deployed has checksum",
			state: &ProjectState{
				State: StateDeployed,
				Data: DeployedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "deployed_checksum",
					DeployedAt:  time.Now().UTC(),
				},
			},
			expectedSum:    "deployed_checksum",
			expectedExists: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checksum, exists := tt.state.Checksum()
			assert.Equal(t, tt.expectedExists, exists)
			assert.Equal(t, tt.expectedSum, checksum)
		})
	}
}

func TestProjectState_IsValidated(t *testing.T) {
	tests := []struct {
		name     string
		state    *ProjectState
		expected bool
	}{
		{
			name:     "scaffolded is not validated",
			state:    NewProjectState(),
			expected: false,
		},
		{
			name: "validated is validated",
			state: &ProjectState{
				State: StateValidated,
				Data: ValidatedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "abc123",
				},
			},
			expected: true,
		},
		{
			name: "deployed is validated",
			state: &ProjectState{
				State: StateDeployed,
				Data: DeployedData{
					ValidatedAt: time.Now().UTC(),
					Checksum:    "abc123",
					DeployedAt:  time.Now().UTC(),
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.IsValidated())
		})
	}
}

func TestLoadState_NotExists(t *testing.T) {
	tempDir := t.TempDir()

	state, err := LoadState(tempDir)
	require.NoError(t, err)
	assert.Nil(t, state)
}

func TestSaveAndLoadState_Roundtrip(t *testing.T) {
	tempDir := t.TempDir()

	original := &ProjectState{
		State: StateValidated,
		Data: ValidatedData{
			ValidatedAt: time.Now().UTC().Truncate(time.Second),
			Checksum:    "test_checksum_123",
		},
	}

	err := SaveState(tempDir, original)
	require.NoError(t, err)

	statePath := filepath.Join(tempDir, StateFileName)
	_, err = os.Stat(statePath)
	require.NoError(t, err)

	loaded, err := LoadState(tempDir)
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, StateValidated, loaded.State)

	dataMap, ok := loaded.Data.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, "test_checksum_123", dataMap["checksum"])
}

func TestComputeChecksum(t *testing.T) {
	tempDir := t.TempDir()

	clientDir := filepath.Join(tempDir, "client")
	serverDir := filepath.Join(tempDir, "server")
	err := os.MkdirAll(clientDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(serverDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(clientDir, "index.ts"), []byte("console.log('client');"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(serverDir, "main.ts"), []byte("console.log('server');"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name":"test"}`), 0644)
	require.NoError(t, err)

	checksum1, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum1)

	checksum2, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2, "checksums should be deterministic")

	err = os.WriteFile(filepath.Join(clientDir, "index.ts"), []byte("console.log('modified');"), 0644)
	require.NoError(t, err)

	checksum3, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3, "checksum should change when files are modified")
}

func TestComputeChecksum_EmptyProject(t *testing.T) {
	tempDir := t.TempDir()

	_, err := ComputeChecksum(tempDir)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no files to hash")
}

func TestComputeChecksum_ExcludesDirectories(t *testing.T) {
	tempDir := t.TempDir()

	clientDir := filepath.Join(tempDir, "client")
	nodeModulesDir := filepath.Join(clientDir, "node_modules")
	distDir := filepath.Join(clientDir, "dist")

	err := os.MkdirAll(nodeModulesDir, 0755)
	require.NoError(t, err)
	err = os.MkdirAll(distDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(clientDir, "index.ts"), []byte("client code"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(nodeModulesDir, "lib.js"), []byte("dependency"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(distDir, "bundle.js"), []byte("compiled"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{"name":"test"}`), 0644)
	require.NoError(t, err)

	checksum, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.NotEmpty(t, checksum)

	err = os.WriteFile(filepath.Join(nodeModulesDir, "another.js"), []byte("more deps"), 0644)
	require.NoError(t, err)

	checksum2, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.Equal(t, checksum, checksum2, "checksum should not change when excluded files are modified")
}

func TestComputeChecksum_FileTypeFiltering(t *testing.T) {
	tempDir := t.TempDir()

	clientDir := filepath.Join(tempDir, "client")
	err := os.MkdirAll(clientDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(clientDir, "file.ts"), []byte("typescript"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(clientDir, "file.txt"), []byte("text"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(clientDir, "file.log"), []byte("log"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{}`), 0644)
	require.NoError(t, err)

	checksum1, err := ComputeChecksum(tempDir)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(clientDir, "file.txt"), []byte("modified text"), 0644)
	require.NoError(t, err)

	checksum2, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.Equal(t, checksum1, checksum2, "checksum should not change for non-source files")

	err = os.WriteFile(filepath.Join(clientDir, "file.ts"), []byte("modified typescript"), 0644)
	require.NoError(t, err)

	checksum3, err := ComputeChecksum(tempDir)
	require.NoError(t, err)
	assert.NotEqual(t, checksum1, checksum3, "checksum should change for source files")
}

func TestVerifyChecksum(t *testing.T) {
	tempDir := t.TempDir()

	clientDir := filepath.Join(tempDir, "client")
	err := os.MkdirAll(clientDir, 0755)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(clientDir, "index.ts"), []byte("code"), 0644)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempDir, "package.json"), []byte(`{}`), 0644)
	require.NoError(t, err)

	checksum, err := ComputeChecksum(tempDir)
	require.NoError(t, err)

	match, err := VerifyChecksum(tempDir, checksum)
	require.NoError(t, err)
	assert.True(t, match)

	match, err = VerifyChecksum(tempDir, "wrong_checksum")
	require.NoError(t, err)
	assert.False(t, match)
}
