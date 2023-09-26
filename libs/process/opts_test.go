package process

import (
	"context"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithEnvs(t *testing.T) {
	if runtime.GOOS != "windows" {
		// Skipping test on windows for now because of the following error:
		// /bin/sh -c echo $FOO $BAR:  exec: "/bin/sh": file does not exist
		t.SkipNow()
	}
	ctx := context.Background()
	res, err := Background(ctx, []string{"/bin/sh", "-c", "echo $FOO $BAR"}, WithEnvs(map[string]string{
		"FOO": "foo",
		"BAR": "delirium",
	}))
	assert.NoError(t, err)
	assert.Equal(t, "foo delirium", res)
}
