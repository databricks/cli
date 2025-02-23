package process

import (
	"bytes"
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestForwarded(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	err := Forwarded(ctx, []string{
		"python3", "-c", "print(input('input: '))",
	}, strings.NewReader("abc\n"), &buf, &buf)
	assert.NoError(t, err)

	assert.Equal(t, "input: abc", strings.TrimSpace(buf.String()))
}

func TestForwardedFails(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	err := Forwarded(ctx, []string{
		"_non_existent_",
	}, strings.NewReader("abc\n"), &buf, &buf)
	assert.Error(t, err)
}

func TestForwardedFailsOnStdinPipe(t *testing.T) {
	ctx := context.Background()
	var buf bytes.Buffer
	err := Forwarded(ctx, []string{
		"_non_existent_",
	}, strings.NewReader("abc\n"), &buf, &buf, func(_ context.Context, c *exec.Cmd) error {
		c.Stdin = strings.NewReader("x")
		return nil
	})
	assert.Error(t, err)
}
