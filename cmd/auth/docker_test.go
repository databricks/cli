package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerTokenCommandIsVisibleAndExperimental(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "auth", args: []string{"--help"}, want: "  docker "},
		{name: "docker", args: []string{"docker", "--help"}, want: "  token "},
		{name: "token", args: []string{"docker", "token", "--help"}, want: "Experimental"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := New()
			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetArgs(tt.args)

			require.NoError(t, cmd.Execute())
			assert.Contains(t, stdout.String(), tt.want)
		})
	}
}
