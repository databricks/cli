package genieclicmd

import (
	"testing"

	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/execv"
	"github.com/databricks/cli/libs/process"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureArgs(t *testing.T) {
	tests := []struct {
		name string
		cfg  *config.Config
		want []string
	}{
		{
			name: "pat profile uses --use-pat",
			cfg:  &config.Config{Profile: "prod", AuthType: auth.AuthTypePat},
			want: []string{"ucode", "configure", "--agents", "codex", "--skip-validate", "--skip-upgrade", "--profiles", "prod", "--use-pat"},
		},
		{
			name: "oauth profile omits --use-pat",
			cfg:  &config.Config{Profile: "prod", AuthType: "databricks-cli"},
			want: []string{"ucode", "configure", "--agents", "codex", "--skip-validate", "--skip-upgrade", "--profiles", "prod"},
		},
		{
			name: "host without profile falls back to --workspaces",
			cfg:  &config.Config{Host: "https://myworkspace.databricks.com"},
			want: []string{"ucode", "configure", "--agents", "codex", "--skip-validate", "--skip-upgrade", "--workspaces", "https://myworkspace.databricks.com"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := configureArgs("ucode", tt.cfg)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigureArgsNoIdentity(t *testing.T) {
	_, err := configureArgs("ucode", &config.Config{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "databricks auth login")
}

func TestEnsureUcodeAlreadyOnPath(t *testing.T) {
	restore := lookPath
	t.Cleanup(func() { lookPath = restore })
	lookPath = func(file string) (string, error) {
		assert.Equal(t, ucodeBin, file)
		return "/usr/local/bin/ucode", nil
	}

	path, err := ensureUcode(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "/usr/local/bin/ucode", path)
}

func TestEnsureUcodeMissingUvErrors(t *testing.T) {
	restore := lookPath
	t.Cleanup(func() { lookPath = restore })
	lookPath = func(file string) (string, error) {
		return "", assert.AnError
	}

	_, err := ensureUcode(t.Context())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "uv")
}

func TestEnsureUcodeInstallsWhenMissing(t *testing.T) {
	restore := lookPath
	t.Cleanup(func() { lookPath = restore })
	// ucode absent on the first probe, uv present, ucode present after install.
	seen := map[string]int{}
	lookPath = func(file string) (string, error) {
		seen[file]++
		switch file {
		case uvBin:
			return "/usr/local/bin/uv", nil
		case ucodeBin:
			if seen[ucodeBin] == 1 {
				return "", assert.AnError
			}
			return "/home/user/.local/bin/ucode", nil
		}
		return "", assert.AnError
	}

	ctx, stub := process.WithStub(t.Context())
	stub.WithStdoutFor("uv tool install", "installed")

	path, err := ensureUcode(ctx)
	require.NoError(t, err)
	assert.Equal(t, "/home/user/.local/bin/ucode", path)
	assert.Len(t, stub.Commands(), 1)
	assert.Contains(t, stub.Commands()[0], "uv tool install")
}

func TestLaunchAgentPassesThroughArgs(t *testing.T) {
	restore := launch
	t.Cleanup(func() { launch = restore })
	var gotArgs []string
	launch = func(opts execv.Options) error {
		gotArgs = opts.Args
		return nil
	}

	err := launchAgent(t.Context(), "/usr/local/bin/ucode", []string{"--full-auto"})
	require.NoError(t, err)
	assert.Equal(t, []string{"/usr/local/bin/ucode", "codex", "--skip-preflight", "--full-auto"}, gotArgs)
}
