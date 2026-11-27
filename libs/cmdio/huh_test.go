package cmdio_test

import (
	"bytes"
	"strings"
	"testing"

	"charm.land/huh/v2"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunHuhFieldUsesContextIO(t *testing.T) {
	var stdout, stderr bytes.Buffer
	ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("hello\r"), &stdout, &stderr))

	var value string
	err := cmdio.RunHuhField(ctx, huh.NewInput().Title("Name").Value(&value))
	require.NoError(t, err)

	// The answer is read from the context's input and the program draws on its
	// stderr. The form quits before its first frame, so only the terminal
	// setup sequences reach stderr.
	assert.Equal(t, "hello", value)
	assert.Empty(t, stdout.String())
	assert.NotEmpty(t, stderr.String())
}
