package cmdio_test

import (
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
)

func TestLogProgressWritesWhenNotQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	cmdio.LogProgress(ctx, "Uploading bundle files")

	assert.Contains(t, stderr.String(), "Uploading bundle files")
}

func TestLogProgressSuppressedWhenQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdio.WithQuiet(ctx)

	cmdio.LogProgress(ctx, "Uploading bundle files")

	assert.Empty(t, stderr.String())
}

// LogString is for results the user asked for, so -qq must not silence it.
func TestLogStringNotAffectedByQuiet(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())
	ctx = cmdio.WithQuiet(ctx)

	cmdio.LogString(ctx, "Resources: 1 created")

	assert.Contains(t, stderr.String(), "Resources: 1 created")
}

func TestLogStringEscapesControlCharacters(t *testing.T) {
	ctx, stderr := cmdio.NewTestContextWithStderr(t.Context())

	cmdio.LogString(ctx, "\x1b[32mRun\x1b[0m failed: no such file \"a\x1b]0;title\x07\x1b[2J\rb\"\nsecond line")

	assert.Equal(t, "\x1b[32mRun\x1b[0m"+` failed: no such file "a\x1b]0;title\a\x1b[2J\rb"`+"\nsecond line\n", stderr.String())
}

func TestIsQuiet(t *testing.T) {
	ctx := t.Context()
	assert.False(t, cmdio.IsQuiet(ctx))
	assert.True(t, cmdio.IsQuiet(cmdio.WithQuiet(ctx)))
}
