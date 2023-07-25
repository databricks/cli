package template

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
