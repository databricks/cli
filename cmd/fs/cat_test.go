package fs

import (
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type trackingReadCloser struct {
	io.Reader
	closed bool
}

func (r *trackingReadCloser) Close() error {
	r.closed = true
	return nil
}

func TestRenderFileClosesReader(t *testing.T) {
	reader := &trackingReadCloser{Reader: strings.NewReader("contents")}
	require.NoError(t, renderFile(cmdio.MockDiscard(t.Context()), reader))
	assert.True(t, reader.closed)
}
