package aircmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseOverrides(t *testing.T) {
	tests := []struct {
		name    string
		in      []string
		want    []overrideEntry
		wantErr string
	}{
		{
			name: "key=value pairs preserve order",
			in:   []string{"compute.num_accelerators=8", "timeout_minutes=45"},
			want: []overrideEntry{
				{path: "compute.num_accelerators", raw: "8"},
				{path: "timeout_minutes", raw: "45"},
			},
		},
		{
			name: "value may contain =",
			in:   []string{"env_variables.EXPR=a=b"},
			want: []overrideEntry{{path: "env_variables.EXPR", raw: "a=b"}},
		},
		{
			name: "key is trimmed",
			in:   []string{"  timeout_minutes = 45"},
			want: []overrideEntry{{path: "timeout_minutes", raw: " 45"}},
		},
		{
			name:    "missing = is rejected",
			in:      []string{"compute.num_accelerators"},
			wantErr: `expected KEY=VALUE`,
		},
		{
			name:    "a .yaml token hints at -f",
			in:      []string{"train.yaml"},
			wantErr: `looks like a config file`,
		},
		{
			name:    "empty key is rejected",
			in:      []string{"=5"},
			wantErr: `empty key`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseOverrides(tt.in)
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestValidateOverridePaths(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr string
	}{
		{name: "known top-level field", path: "experiment_name"},
		{name: "known nested field", path: "compute.num_accelerators"},
		{name: "free-form sub-path", path: "env_variables.MY_VAR"},
		{name: "deep free-form sub-path", path: "parameters.model.layers"},
		{
			name:    "unknown top-level field",
			path:    "bogus",
			wantErr: `"bogus" is not a known field`,
		},
		{
			name:    "unknown nested field",
			path:    "compute.bogus",
			wantErr: `"bogus" is not a known field`,
		},
		{
			name:    "sub-field of a scalar",
			path:    "command.sub",
			wantErr: `"command" is not a nested object`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateOverridePaths([]overrideEntry{{path: tt.path, raw: "x"}})
			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
		})
	}
}

// overrideBaseConfig is a valid 8-GPU config the override tests mutate.
const overrideBaseConfig = `experiment_name: smoke
command: python train.py
compute:
  accelerator_type: GPU_8xH100
  num_accelerators: 8
env_variables:
  EXISTING: hello
`

func TestLoadRunConfigWithOverrides(t *testing.T) {
	t.Run("no overrides matches loadRunConfig", func(t *testing.T) {
		cfg, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), nil)
		require.NoError(t, err)
		assert.Equal(t, 8, cfg.Compute.NumAccelerators)
	})

	t.Run("typed scalar override is coerced", func(t *testing.T) {
		cfg, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"compute.num_accelerators=16"})
		require.NoError(t, err)
		assert.Equal(t, 16, cfg.Compute.NumAccelerators)
	})

	t.Run("multiple overrides all apply", func(t *testing.T) {
		cfg, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"compute.num_accelerators=16", "timeout_minutes=45"})
		require.NoError(t, err)
		assert.Equal(t, 16, cfg.Compute.NumAccelerators)
		require.NotNil(t, cfg.TimeoutMinutes)
		assert.Equal(t, 45, *cfg.TimeoutMinutes)
	})

	t.Run("free-form env var adds a key as a string", func(t *testing.T) {
		cfg, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"env_variables.RANK=0"})
		require.NoError(t, err)
		// A numeric-looking value stays a string because env_variables is map[string]string.
		assert.Equal(t, "0", cfg.EnvVariables["RANK"])
	})

	t.Run("intermediate maps are auto-created", func(t *testing.T) {
		cfg, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"environment.docker_image.url=my/img:1"})
		require.NoError(t, err)
		require.NotNil(t, cfg.Environment)
		require.NotNil(t, cfg.Environment.DockerImage)
		assert.Equal(t, "my/img:1", cfg.Environment.DockerImage.URL)
	})

	t.Run("unknown path errors before mutation", func(t *testing.T) {
		_, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"bogus=1"})
		require.ErrorContains(t, err, `"bogus" is not a known field`)
	})

	t.Run("semantic validation runs after override", func(t *testing.T) {
		// 3 is a known field with a valid type, so only validate() can reject it.
		_, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"compute.num_accelerators=3"})
		require.ErrorContains(t, err, "must be a multiple of 8")
	})

	t.Run("type mismatch is rejected on re-decode", func(t *testing.T) {
		_, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"compute.num_accelerators=abc"})
		require.Error(t, err)
	})

	t.Run("malformed override is rejected", func(t *testing.T) {
		_, err := loadRunConfigWithOverrides(t.Context(), writeConfig(t, overrideBaseConfig), []string{"compute.num_accelerators"})
		require.ErrorContains(t, err, "expected KEY=VALUE")
	})
}
