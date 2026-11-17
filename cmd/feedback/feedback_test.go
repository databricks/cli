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

const commandHint = `; --command takes a command path such as "bundle deploy", without the databricks prefix, flags or arguments`

// newTestFeedbackCommand returns the feedback command under a root with the
// root's bundle target flags and a few commands --command can name.
func newTestFeedbackCommand() *cobra.Command {
	rootCmd := &cobra.Command{Use: "databricks"}
	rootCmd.PersistentFlags().String("target", "", "")
	rootCmd.PersistentFlags().String("environment", "", "")
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
	return feedbackCmd
}

func TestResolveCommand(t *testing.T) {
	feedbackCmd := newTestFeedbackCommand()

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
		{name: "unknown top level", command: "unknown", wantErr: `unknown command "unknown"` + commandHint},
		{name: "unknown subcommand", command: "bundle unknown", wantErr: `unknown command "bundle unknown"` + commandHint},
		{name: "with arguments", command: "bundle deploy -t dev", wantErr: `unknown command "bundle deploy -t dev"` + commandHint},
		{name: "blank", command: " ", wantErr: `unknown command " "` + commandHint},
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

func TestNewSubmission(t *testing.T) {
	tests := []struct {
		name    string
		flags   []string
		opts    options
		message string
		want    submission
		wantErr string
	}{
		{
			name:    "all fields",
			opts:    options{command: "bundle  deploy", runID: "+00123", canBeContacted: true},
			message: " slow ",
			want: submission{
				text: "slow", command: "bundle deploy", component: "DeveloperEcosystem.CoreDabs",
				origin: "bundle_deploy", runID: "123", canBeContacted: true,
			},
		},
		{name: "zero run ID", opts: options{runID: "0"}, message: "x", wantErr: "--run-id must be a positive job run ID"},
		{name: "non-numeric run ID", opts: options{runID: "abc"}, message: "x", wantErr: "--run-id must be a positive job run ID"},
		{name: "blank message", message: " ", wantErr: "feedback message must contain 1 to 10000 characters"},
		{name: "unknown command", opts: options{command: "nope"}, message: "x", wantErr: `unknown command "nope"` + commandHint},
		{
			name: "target", flags: []string{"--target", "dev"}, message: "x",
			wantErr: "--target selects a bundle target, which feedback does not use; use --profile to choose the workspace",
		},
		{
			name: "deprecated environment", flags: []string{"--environment", "dev"}, message: "x",
			wantErr: "--environment selects a bundle target, which feedback does not use; use --profile to choose the workspace",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			feedbackCmd := newTestFeedbackCommand()
			require.NoError(t, feedbackCmd.ParseFlags(tt.flags))
			got, err := newSubmission(feedbackCmd, tt.message, &tt.opts)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name string
		s    submission
		want string
	}{
		{
			name: "message only",
			s:    submission{text: "line one\n\tline two"},
			want: "line one\n\tline two\n\nDatabricks may contact you: no\n",
		},
		{
			name: "all fields",
			s:    submission{text: "slow", command: "air run", runID: "42", canBeContacted: true},
			want: "slow\n\nCommand: air run\nRun ID: 42\nDatabricks may contact you: yes\n",
		},
		{
			name: "control characters cannot hide text",
			s:    submission{text: "great\x1b[8m hidden\x1b[0m\r\u0085"},
			want: `great\x1b[8m hidden\x1b[0m\r\u0085` + "\n\nDatabricks may contact you: no\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, summary(tt.s))
		})
	}
}

func TestConfirm(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("y\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, submission{text: "exact feedback"})
		require.NoError(t, err)
		assert.True(t, confirmed)
		assert.Contains(t, stderr.String(), "The following feedback will be sent to Databricks:\n\nexact feedback\n\nDatabricks may contact you: no\n\n")
		assert.Contains(t, stderr.String(), "Send this feedback? [y/N]: ")
	})

	t.Run("declined", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, submission{text: "exact feedback"})
		require.NoError(t, err)
		assert.False(t, confirmed)
		assert.Contains(t, stderr.String(), "Feedback not sent.")
	})

	t.Run("no terminal", func(t *testing.T) {
		testutil.SetAgent(t, "test-agent")
		ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), flags.OutputText, strings.NewReader(""), io.Discard, io.Discard, "", ""))
		confirmed, err := confirm(ctx, submission{text: "exact feedback"})
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
			confirmed, err := confirm(ctx, submission{text: "exact feedback"})
			require.NoError(t, err)
			assert.True(t, confirmed)
		})
	}
}
