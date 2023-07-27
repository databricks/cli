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

	cleanContent := strings.Trim(string(r.files[0].content), "\n\r")
	assert.Equal(t, `{{ fail "abc" }}`, cleanContent)
}

func TestTemplateRegexpCompileFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	r, err := newRenderer(ctx, nil, "./testdata/regexp-compile/template", "./testdata/regexp-compile/library", tmpDir)
	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	content := string(r.files[0].content)
	assert.Contains(t, content, "0:food")
	assert.Contains(t, content, "1:fool")
}

func TestTemplateUrlFunction(t *testing.T) {
	ctx := context.Background()
	tmpDir := t.TempDir()

	r, err := newRenderer(ctx, nil, "./testdata/urlparse-function/template", "./testdata/urlparse-function/library", tmpDir)

	require.NoError(t, err)

	err = r.walk()
	assert.NoError(t, err)

	assert.Len(t, r.files, 1)
	cleanContent := strings.Trim(string(r.files[0].content), "\n\r")

	assert.Equal(t, "https://www.databricks.com", cleanContent)
}
