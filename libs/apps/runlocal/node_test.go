package runlocal

import (
	"testing"

	"github.com/databricks/cli/libs/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNodeAppGetCommand(t *testing.T) {
	tests := []struct {
		name    string
		debug   bool
		command []string
		wantCmd []string
		wantEnv []string
	}{
		{
			name:    "default command without debug",
			wantCmd: []string{"npm", "run", "start"},
		},
		{
			name:    "custom command without debug",
			command: []string{"node", "server.js"},
			wantCmd: []string{"node", "server.js"},
		},
		{
			name:    "default command with debug",
			debug:   true,
			wantCmd: []string{"npm", "run", "start"},
			wantEnv: []string{"NODE_OPTIONS=--inspect=9229"},
		},
		{
			name:    "custom command with debug",
			debug:   true,
			command: []string{"node", "server.js"},
			wantCmd: []string{"node", "server.js"},
			wantEnv: []string{"NODE_OPTIONS=--inspect=9229"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &Config{}
			spec := &AppSpec{config: config, Command: tt.command}
			app := NewNodeApp(config, spec, &PackageJson{})
			cmd, cmdEnv, err := app.GetCommand(t.Context(), tt.debug)
			require.NoError(t, err)
			assert.Equal(t, tt.wantCmd, cmd)
			assert.Equal(t, tt.wantEnv, cmdEnv)
		})
	}
}

func TestNodeAppGetCommandDebugAppendsToExistingNodeOptions(t *testing.T) {
	ctx := t.Context()
	ctx = env.Set(ctx, "NODE_OPTIONS", "--max-old-space-size=4096")

	config := &Config{}
	spec := &AppSpec{config: config}
	app := NewNodeApp(config, spec, &PackageJson{})
	_, cmdEnv, err := app.GetCommand(ctx, true)
	require.NoError(t, err)
	assert.Equal(t, []string{"NODE_OPTIONS=--max-old-space-size=4096 --inspect=9229"}, cmdEnv)
}

func TestNodeAppGetCommandDebugCustomPort(t *testing.T) {
	config := &Config{DebugPort: "5555"}
	spec := &AppSpec{config: config}
	app := NewNodeApp(config, spec, &PackageJson{})
	_, cmdEnv, err := app.GetCommand(t.Context(), true)
	require.NoError(t, err)
	assert.Equal(t, []string{"NODE_OPTIONS=--inspect=5555"}, cmdEnv)
}
