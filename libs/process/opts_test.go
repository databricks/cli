package process

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithEnvs(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{"/bin/sh", "-c", "echo $FOO $BAR"}, WithEnvs(map[string]string{
		"FOO": "foo",
		"BAR": "delirium",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "foo delirium", res)
}
