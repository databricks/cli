package auth

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQuoteEnvValue(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{name: "simple value", in: "hello", want: "hello"},
		{name: "empty value", in: "", want: `''`},
		{name: "value with space", in: "hello world", want: "'hello world'"},
		{name: "value with tab", in: "hello\tworld", want: "'hello\tworld'"},
		{name: "value with double quote", in: `say "hi"`, want: "'say \"hi\"'"},
		{name: "value with backslash", in: `path\to`, want: "'path\\to'"},
		{name: "url value", in: "https://example.com", want: "https://example.com"},
		{name: "value with dollar", in: "price$5", want: "'price$5'"},
		{name: "value with backtick", in: "hello`world", want: "'hello`world'"},
		{name: "value with bang", in: "hello!world", want: "'hello!world'"},
		{name: "value with single quote", in: "it's", want: "'it'\\''s'"},
		{name: "value with newline", in: "line1\nline2", want: "'line1\nline2'"},
		{name: "value with carriage return", in: "line1\rline2", want: "'line1\rline2'"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, quoteEnvValue(c.in))
		})
	}
}

func TestNewEnvCommandDeprecation(t *testing.T) {
	cmd := newEnvCommand()
	assert.True(t, cmd.Hidden, "env command must remain hidden")
	assert.Contains(t, cmd.Long, "Deprecated", "Long description should mention deprecation")
	assert.Contains(t, envDeprecationWarning, "deprecated")
	assert.Contains(t, envDeprecationWarning, "databricks auth env")
}

func TestWriteEnvOutput(t *testing.T) {
	envVars := map[string]string{
		"DATABRICKS_HOST":  "https://test.cloud.databricks.com",
		"DATABRICKS_TOKEN": "secret value",
	}

	t.Run("text mode emits sorted shell-quoted KEY=VALUE lines", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, writeEnvOutput(&buf, envVars, true))
		assert.Equal(t, "DATABRICKS_HOST=https://test.cloud.databricks.com\nDATABRICKS_TOKEN='secret value'\n", buf.String())
	})

	t.Run("json mode wraps env in {\"env\": ...} with trailing newline", func(t *testing.T) {
		var buf bytes.Buffer
		require.NoError(t, writeEnvOutput(&buf, envVars, false))
		assert.Equal(t, "{\n  \"env\": {\n    \"DATABRICKS_HOST\": \"https://test.cloud.databricks.com\",\n    \"DATABRICKS_TOKEN\": \"secret value\"\n  }\n}\n", buf.String())
	})

	t.Run("empty map", func(t *testing.T) {
		var textBuf, jsonBuf bytes.Buffer
		require.NoError(t, writeEnvOutput(&textBuf, map[string]string{}, true))
		require.NoError(t, writeEnvOutput(&jsonBuf, map[string]string{}, false))
		assert.Empty(t, textBuf.String())
		assert.Equal(t, "{\n  \"env\": {}\n}\n", jsonBuf.String())
	})
}
