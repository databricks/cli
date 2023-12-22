package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArtifactBuild(t *testing.T) {
	artifact := Artifact{
		BuildCommand: "echo 'Hello from build command'",
	}
	res, err := artifact.Build(context.Background())
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "Hello from build command\n", string(res))
}
