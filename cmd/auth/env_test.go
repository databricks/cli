package auth

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
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
		{name: "empty value", in: "", want: `""`},
		{name: "value with space", in: "hello world", want: `"hello world"`},
		{name: "value with tab", in: "hello\tworld", want: "\"hello\tworld\""},
		{name: "value with double quote", in: `say "hi"`, want: `"say \"hi\""`},
		{name: "value with backslash", in: `path\to`, want: `"path\\to"`},
		{name: "url value", in: "https://example.com", want: "https://example.com"},
		{name: "value with dollar", in: "price$5", want: `"price\$5"`},
		{name: "value with backtick", in: "hello`world", want: `"hello\` + "`" + `world"`},
		{name: "value with bang", in: "hello!world", want: `"hello\!world"`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := quoteEnvValue(c.in)
			assert.Equal(t, c.want, got)
		})
	}
}

func TestEnvCommand_TextOutput(t *testing.T) {
	cases := []struct {
		name     string
		args     []string
		wantJSON bool
	}{
		{
			name:     "default output is JSON",
			args:     []string{"--host", "https://test.cloud.databricks.com"},
			wantJSON: true,
		},
		{
			name:     "explicit --output text produces KEY=VALUE lines",
			args:     []string{"--host", "https://test.cloud.databricks.com", "--output", "text"},
			wantJSON: false,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			parent := &cobra.Command{Use: "databricks"}
			outputFlag := flags.OutputText
			parent.PersistentFlags().VarP(&outputFlag, "output", "o", "output type: text or json")

			envCmd := newEnvCommand()
			parent.AddCommand(envCmd)

			// Set DATABRICKS_TOKEN so the SDK's config.Authenticate succeeds
			// without hitting a real endpoint.
			t.Setenv("DATABRICKS_TOKEN", "test-token-value")

			var buf bytes.Buffer
			parent.SetOut(&buf)
			parent.SetArgs(append([]string{"env"}, c.args...))

			err := parent.Execute()
			require.NoError(t, err)

			output := buf.String()
			if c.wantJSON {
				assert.Contains(t, output, "{")
				assert.Contains(t, output, "DATABRICKS_HOST")
			} else {
				assert.NotContains(t, output, "{")
				assert.Contains(t, output, "DATABRICKS_HOST=")
				assert.Contains(t, output, "=")
				// Verify KEY=VALUE format (no JSON structure)
				assert.NotContains(t, output, `"env"`)
			}
		})
	}
}
