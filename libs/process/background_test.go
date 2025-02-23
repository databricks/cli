package process

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func splitLines(b []byte) (lines []string) {
	scan := bufio.NewScanner(bytes.NewReader(b))
	for scan.Scan() {
		line := scan.Text()
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
}

func TestBackgroundUnwrapsNotFound(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"meeecho", "1"})
	assert.ErrorIs(t, err, exec.ErrNotFound)
}

func TestBackground(t *testing.T) {
	ctx := context.Background()
	res, err := Background(ctx, []string{"echo", "1"}, WithDir("/"))
	assert.NoError(t, err)
	assert.Equal(t, "1", strings.TrimSpace(res))
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
	buf := bytes.Buffer{}
	res, err := Background(ctx, []string{
		"python3", "-c", "import sys, time; " +
			`sys.stderr.write("1\n"); sys.stderr.flush(); ` +
			"time.sleep(0.001); " +
			"print('2', flush=True); sys.stdout.flush(); " +
			"time.sleep(0.001)",
	}, WithCombinedOutput(&buf))
	assert.NoError(t, err)
	assert.Equal(t, "2", strings.TrimSpace(res))

	// The order of stdout and stderr being read into the buffer
	// for combined output is not deterministic due to scheduling
	// of the underlying goroutines that consume them.
	// That's why this asserts on the contents and not the order.
	assert.ElementsMatch(t, []string{"1", "2"}, splitLines(buf.Bytes()))
}

func TestBackgroundCombinedOutputFailure(t *testing.T) {
	ctx := context.Background()
	buf := bytes.Buffer{}
	res, err := Background(ctx, []string{
		"python3", "-c", "import sys, time; " +
			`sys.stderr.write("1\n"); sys.stderr.flush(); ` +
			"time.sleep(0.001); " +
			"print('2', flush=True); sys.stdout.flush(); " +
			"time.sleep(0.001); " +
			"sys.exit(42)",
	}, WithCombinedOutput(&buf))
	var processErr *ProcessError
	if assert.ErrorAs(t, err, &processErr) {
		assert.Equal(t, "1", strings.TrimSpace(processErr.Stderr))
		assert.Equal(t, "2", strings.TrimSpace(processErr.Stdout))
	}
	assert.Equal(t, "2", strings.TrimSpace(res))
	assert.ElementsMatch(t, []string{"1", "2"}, splitLines(buf.Bytes()))
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
	assert.Error(t, err)
}

func TestBackgroundFailsOnOption(t *testing.T) {
	ctx := context.Background()
	_, err := Background(ctx, []string{"ls", "/dev/null/x"}, func(_ context.Context, c *exec.Cmd) error {
		return errors.New("nope")
	})
	assert.EqualError(t, err, "nope")
}
