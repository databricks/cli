package pipelines

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderJSON(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"id", "name"}, [][]string{{"1", "alice"}, {"2", "bob"}})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"id":"1","name":"alice"},{"id":"2","name":"bob"}]`, buf.String())
}

func TestRenderJSONEmpty(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"id"}, nil)
	require.NoError(t, err)
	assert.JSONEq(t, `[]`, buf.String())
}

func TestRenderJSONPreservesColumnOrder(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"name", "id"}, [][]string{{"alice", "1"}})
	require.NoError(t, err)
	assert.Equal(t, "[\n  {\n    \"name\": \"alice\",\n    \"id\": \"1\"\n  }\n]\n", buf.String())
}

func TestRenderJSONEscapesValues(t *testing.T) {
	var buf bytes.Buffer
	err := renderJSON(&buf, []string{"c"}, [][]string{{`a"b	c`}})
	require.NoError(t, err)
	assert.JSONEq(t, `[{"c":"a\"b\tc"}]`, buf.String())
}
