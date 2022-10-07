package api

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestBodyEmpty(t *testing.T) {
	out, err := requestBody("")
	require.NoError(t, err)
	assert.Equal(t, nil, out)
}

func TestRequestBodyString(t *testing.T) {
	out, err := requestBody("foo")
	require.NoError(t, err)
	assert.Equal(t, "foo", out)
}

func TestRequestBodyFile(t *testing.T) {
	var fpath string
	var payload = []byte("hello world\n")

	{
		f, err := os.Create(path.Join(t.TempDir(), "file"))
		require.NoError(t, err)
		f.Write(payload)
		f.Close()
		fpath = f.Name()
	}

	out, err := requestBody(fmt.Sprintf("@%s", fpath))
	require.NoError(t, err)
	assert.Equal(t, string(payload), out)
}
