package structwalk_test

import (
	"encoding/json"
	"testing"

	"github.com/databricks/cli/libs/structs/structwalk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedactSensitiveFields_PlainString(t *testing.T) {
	type Config struct {
		Name  string `json:"name"`
		Token string `json:"token" bundle:"sensitive"`
	}

	src := Config{Name: "my-resource", Token: "s3cr3t"}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got Config
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "my-resource", got.Name)
	assert.Equal(t, "<redacted>", got.Token)
}

func TestRedactSensitiveFields_OriginalUnchanged(t *testing.T) {
	type Config struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	src := Config{Token: "s3cr3t"}
	_, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)
	assert.Equal(t, "s3cr3t", src.Token, "original must not be modified")
}

func TestRedactSensitiveFields_EmptyStringUnchanged(t *testing.T) {
	type Config struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	src := Config{Token: ""}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got Config
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Empty(t, got.Token, "empty sensitive string should stay empty")
}

func TestRedactSensitiveFields_NonSensitiveUnchanged(t *testing.T) {
	type Config struct {
		Name string `json:"name"`
	}

	src := Config{Name: "hello"}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got Config
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "hello", got.Name)
}

func TestRedactSensitiveFields_Pointer(t *testing.T) {
	type Config struct {
		Token string `json:"token" bundle:"sensitive"`
	}

	src := &Config{Token: "s3cr3t"}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got Config
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "<redacted>", got.Token)
	assert.Equal(t, "s3cr3t", src.Token, "original pointer target must not be modified")
}

func TestRedactSensitiveFields_NestedStruct(t *testing.T) {
	type Inner struct {
		Secret string `json:"secret" bundle:"sensitive"`
	}
	type Outer struct {
		Name  string `json:"name"`
		Inner Inner  `json:"inner"`
	}

	src := Outer{Name: "outer", Inner: Inner{Secret: "s3cr3t"}}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got Outer
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "outer", got.Name)
	assert.Equal(t, "<redacted>", got.Inner.Secret)
}

func TestRedactSensitiveFields_Slice(t *testing.T) {
	type Config struct {
		Token string `json:"token" bundle:"sensitive"`
		Name  string `json:"name"`
	}

	src := []Config{{Token: "s1", Name: "a"}, {Token: "s2", Name: "b"}}
	b, err := structwalk.RedactSensitiveFields(src, "<redacted>")
	require.NoError(t, err)

	var got []Config
	require.NoError(t, json.Unmarshal(b, &got))
	assert.Equal(t, "<redacted>", got[0].Token)
	assert.Equal(t, "<redacted>", got[1].Token)
	assert.Equal(t, "a", got[0].Name)
	assert.Equal(t, "b", got[1].Name)
}
