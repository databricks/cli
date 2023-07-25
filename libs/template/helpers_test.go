package template

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplatePrintStringWithoutProcessing(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	r, err := newRenderer(ctx, nil, "./testdata/print-without-processing/template", "./testdata/print-without-processing/library", tmpDir)
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	cleanContent := strings.Trim(string(r.files[0].content), "\n\r")
	assert.Equal(t, `{{ fail "abc" }}`, cleanContent)
}
