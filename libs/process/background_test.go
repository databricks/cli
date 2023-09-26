package process

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBackgroundUnwrapsNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"/bin/meeecho", "1"})
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestBackground(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{"echo", "1"}, WithDir("/"))
	assert.NoError(t, err)
	assert.Equal(t, "1", res)
}

func TestBackgroundOnlyStdoutGetsoutOnSuccess(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{
		"python3", "-c", "import sys; sys.stderr.write('1'); sys.stdout.write('2')",
	})
	assert.NoError(t, err)
	assert.Equal(t, "2", res)
}

func TestBackgroundCombinedOutput(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	res, err := Background(ctx, []string{
		"python3", "-c", "import sys; sys.stderr.write('1'); sys.stdout.write('2')",
	}, WithCombinedOutput(&buf))
	assert.NoError(t, err)
	assert.Equal(t, "2", res)
	assert.Equal(t, "12", buf.String())
}

func TestBackgroundCombinedOutputFailure(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	res, err := Background(ctx, []string{
		"python3", "-c", "import sys; sys.stderr.write('1'); sys.stdout.write('2'); sys.exit(42)",
	}, WithCombinedOutput(&buf))
	var processErr *ProcessError
	if assert.ErrorAs(t, err, &processErr) {
		assert.Equal(t, "1", processErr.Stderr)
		assert.Equal(t, "2", processErr.Stdout)
	}
	assert.Equal(t, "2", res)
	assert.Equal(t, "12", buf.String())
}

func TestBackgroundNoStdin(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{"cat"})
	assert.NoError(t, err)
	assert.Equal(t, "", res)
}

func TestBackgroundFails(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"ls", "/dev/null/x"})
	assert.NotNil(t, err)
}

func TestBackgroundFailsOnOption(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"ls", "/dev/null/x"}, func(_ context.Context, c *exec.Cmd) error {
		return fmt.Errorf("nope")
	})
	assert.EqualError(t, err, "nope")
}
