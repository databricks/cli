package flags

import (
	"fmt"
	"os"
	"path"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJsonFlagEmpty(t *testing.T) {
	var body JsonFlag

	var request any
	err := body.Unmarshal(&request)

	assert.Equal(t, "JSON (0 bytes)", body.String())
	assert.NoError(t, err)
	assert.Nil(t, request)
}

func TestJsonFlagInline(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo": "bar"}`)
	assert.NoError(t, err)

	var request any
	err = body.Unmarshal(&request)
	assert.NoError(t, err)

	assert.Equal(t, "JSON (14 bytes)", body.String())
	assert.Equal(t, map[string]any{"foo": "bar"}, request)
}

func TestJsonFlagError(t *testing.T) {
	var body JsonFlag

	err := body.Set(`{"foo":`)
	assert.NoError(t, err)

	var request any
	err = body.Unmarshal(&request)
	assert.EqualError(t, err, "unexpected end of JSON input")
	assert.Equal(t, "JSON (7 bytes)", body.String())
}

func TestJsonFlagFile(t *testing.T) {
	var body JsonFlag
	var request any

	var fpath string
	var payload = []byte(`"hello world"`)

	{
		f, err := os.Create(path.Join(t.TempDir(), "file"))
		require.NoError(t, err)
		f.Write(payload)
		f.Close()
		fpath = f.Name()
	}

	err := body.Set(fmt.Sprintf("@%s", fpath))
	require.NoError(t, err)

	err = body.Unmarshal(&request)
	require.NoError(t, err)

	assert.Equal(t, "hello world", request)
}

func TestJsonFlagUnmarshal_UnmarshalIgnoredFields(t *testing.T) {
	type Foo struct {
		A string `json:"a"`
		B string `json:"-"`
	}

	raw := `{"a": "foo", "b": "bar"}`
	var body JsonFlag
	body.Set(raw)

	var request Foo
	body.Unmarshal(&request)

	assert.Equal(t, Foo{A: "foo", B: "bar"}, request)
}
