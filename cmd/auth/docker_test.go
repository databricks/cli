package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDockerCommandsAreVisibleAndExperimental(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "auth", args: []string{"--help"}, want: "  docker "},
		{name: "docker", args: []string{"docker", "--help"}, want: "  token "},
		{name: "token", args: []string{"docker", "token", "--help"}, want: "Experimental"},
		{name: "configure listing", args: []string{"docker", "--help"}, want: "  configure "},
		{name: "configure experimental", args: []string{"docker", "configure", "--help"}, want: "Experimental"},
		{name: "configure usage", args: []string{"docker", "configure", "--help"}, want: "auth docker configure [PROFILE]"},
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
