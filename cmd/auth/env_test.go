package auth

import (
	"bytes"
	"testing"

	"github.com/databricks/cli/libs/cmdio"
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
			args:     nil,
			wantJSON: true,
		},
		{
			name:     "explicit --output text produces KEY=VALUE lines",
			args:     []string{"--output", "text"},
			wantJSON: false,
		},
		{
			name:     "explicit --output json produces JSON",
			args:     []string{"--output", "json"},
			wantJSON: true,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			// Isolate from real config/token cache on the machine.
			t.Setenv("DATABRICKS_CONFIG_FILE", t.TempDir()+"/.databrickscfg")
			t.Setenv("HOME", t.TempDir())
			// Set env vars so MustAnyClient resolves auth via PAT.
			t.Setenv("DATABRICKS_HOST", "https://test.cloud.databricks.com")
			t.Setenv("DATABRICKS_TOKEN", "test-token-value")

			parent := &cobra.Command{Use: "databricks"}
			outputFlag := flags.OutputText
			parent.PersistentFlags().VarP(&outputFlag, "output", "o", "output type: text or json")
			parent.PersistentFlags().StringP("profile", "p", "", "~/.databrickscfg profile")

			envCmd := newEnvCommand()
			parent.AddCommand(envCmd)
			parent.SetContext(cmdio.MockDiscard(t.Context()))

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
				assert.NotContains(t, output, `"env"`)
			}
		})
	}
}
