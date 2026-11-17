package feedback

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateMessage(t *testing.T) {
	tests := []struct {
		name    string
		message string
		want    string
		wantErr bool
	}{
		{name: "trimmed", message: "  feedback\n", want: "feedback"},
		{name: "maximum runes", message: strings.Repeat("é", messageLimit), want: strings.Repeat("é", messageLimit)},
		{name: "blank", message: " \n", wantErr: true},
		{name: "too long", message: strings.Repeat("x", messageLimit+1), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateMessage(tt.message)
			if tt.wantErr {
				assert.EqualError(t, err, "feedback message must contain 1 to 10000 characters")
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestResolveCommand(t *testing.T) {
	rootCmd := &cobra.Command{Use: "databricks"}
	feedbackCmd := &cobra.Command{Use: "feedback"}
	rootCmd.AddCommand(feedbackCmd)
	for _, path := range []string{"air run", "bundle deploy", "version"} {
		parent := rootCmd
		for name := range strings.FieldsSeq(path) {
			child := &cobra.Command{Use: name}
			parent.AddCommand(child)
			parent = child
		}
	}

	tests := []struct {
		name          string
		command       string
		wantComponent string
		wantOrigin    string
		wantErr       string
	}{
		{name: "general", wantComponent: defaultComponent, wantOrigin: "cli"},
		{name: "air subcommand", command: "air run", wantComponent: "ModelTraining.AIRuntime", wantOrigin: "air_run"},
		{name: "bundle group", command: "bundle", wantComponent: "DeveloperEcosystem.CoreDabs", wantOrigin: "bundle"},
		{name: "bundle subcommand", command: "bundle  deploy", wantComponent: "DeveloperEcosystem.CoreDabs", wantOrigin: "bundle_deploy"},
		{name: "other command", command: "version", wantComponent: defaultComponent, wantOrigin: "version"},
		{name: "unknown top level", command: "unknown", wantErr: `unknown command "unknown"`},
		{name: "unknown subcommand", command: "bundle unknown", wantErr: `unknown command "bundle unknown"`},
		{name: "blank", command: " ", wantErr: `unknown command " "`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, origin, err := resolveCommand(feedbackCmd, tt.command)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantComponent, component)
			assert.Equal(t, tt.wantOrigin, origin)
		})
	}
}

func TestConfirm(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("y\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, "exact feedback")
		require.NoError(t, err)
		assert.True(t, confirmed)
		assert.Contains(t, stderr.String(), "The following feedback will be sent to Databricks:\n\nexact feedback\n")
		assert.Contains(t, stderr.String(), "Send this feedback? [y/N]: ")
	})

	t.Run("declined", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, "exact feedback")
		require.NoError(t, err)
		assert.False(t, confirmed)
		assert.Contains(t, stderr.String(), "Feedback not sent.")
	})

	t.Run("no terminal", func(t *testing.T) {
		testutil.SetAgent(t, "test-agent")
		ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), io.Discard, io.Discard, "", ""))
		confirmed, err := confirm(ctx, "exact feedback")
		assert.False(t, confirmed)
		assert.EqualError(t, err, "feedback from an AI agent must be confirmed by the user; run with --yes once the user approves the message\n\n"+
			"Note for AI agents (test-agent): show the user the exact feedback message and pass --yes only after the user approves sending it")
	})
}

func TestConfirmWithoutColorSupport(t *testing.T) {
	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{name: "no color", key: "NO_COLOR", value: "1"},
		{name: "dumb terminal", key: "TERM", value: "dumb"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := env.Set(t.Context(), tc.key, tc.value)
			var stderr bytes.Buffer
			in := cmdio.FakeTTYReader(strings.NewReader("y\n"))
			ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, in, io.Discard, cmdio.FakeTTY(&stderr), "", ""))
			assert.False(t, cmdio.IsPromptSupported(ctx))
			confirmed, err := confirm(ctx, "exact feedback")
			require.NoError(t, err)
			assert.True(t, confirmed)
		})
	}
}
