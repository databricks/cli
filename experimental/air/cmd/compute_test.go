package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseGPUType(t *testing.T) {
	tests := []struct {
		in   string
		want gpuType
	}{
		{"GPU_1xA10", gpuType1xA10},
		{"GPU_8xH100", gpuType8xH100},
		{"GPU_1xH100", gpuType1xH100},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := parseGPUType(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestParseGPUTypeInvalid(t *testing.T) {
	// Wrong casing is rejected rather than fixed up; legacy types (h100_80gb, a10)
	// can no longer be submitted; unknown types are rejected.
	for _, in := range []string{"gpu_1xa10", "GPU_1XA10", "GPU_2xH100", "h100_80gb", "a10", "b200", ""} {
		t.Run(in, func(t *testing.T) {
			_, err := parseGPUType(in)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "valid types are")
		})
	}
}

func TestGPUsPerNode(t *testing.T) {
	tests := []struct {
		in   gpuType
		want int
	}{
		{gpuType1xA10, 1},
		{gpuType1xH100, 1},
		{gpuType8xH100, 8},
	}
	for _, tt := range tests {
		t.Run(string(tt.in), func(t *testing.T) {
			got, err := gpusPerNode(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}

	_, err := gpusPerNode(gpuType("nonsense"))
	require.Error(t, err)
}

func TestComputeConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		cfg     computeConfig
		wantErr string // substring; empty means the config is valid
	}{
		{"single node", computeConfig{NumAccelerators: 8, AcceleratorType: "GPU_8xH100"}, ""},
		{"multiple nodes", computeConfig{NumAccelerators: 16, AcceleratorType: "GPU_8xH100"}, ""},
		{"single-gpu partitions", computeConfig{NumAccelerators: 3, AcceleratorType: "GPU_1xH100"}, ""},
		{"unknown type", computeConfig{NumAccelerators: 8, AcceleratorType: "b200"}, "accelerator_type"},
		{"legacy type rejected", computeConfig{NumAccelerators: 8, AcceleratorType: "h100_80gb"}, "accelerator_type"},
		{"non-positive count", computeConfig{NumAccelerators: 0, AcceleratorType: "GPU_1xH100"}, "must be positive"},
		{"count not a multiple", computeConfig{NumAccelerators: 4, AcceleratorType: "GPU_8xH100"}, "multiple of 8"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.validate()
			if tt.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
