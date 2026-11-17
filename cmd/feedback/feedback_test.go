package feedback

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/env"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
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
		{name: "only invisible characters", message: "\u200b\ufe0f\u3164", wantErr: true},
		{name: "visible after invisible", message: "\u200bx", want: "\u200bx"},
		{name: "too long", message: strings.Repeat("x", messageLimit+1), wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := validateMessage(tt.message)
			if tt.wantErr {
				assert.EqualError(t, err, messageError)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

const (
	messageError = "feedback message must contain 1 to 10000 characters, including a visible one"
	runIDError   = "--run-id must be a positive job run ID"
)

const testWorkspace = "https://workspace.test"

const commandHint = `; --command takes a command path such as "bundle deploy", without the databricks prefix, flags or arguments`

// newTestFeedbackCommand returns the feedback command under a root with the
// root's bundle target flags and a few commands --command can name.
func newTestFeedbackCommand() *cobra.Command {
	rootCmd := &cobra.Command{Use: "databricks"}
	rootCmd.PersistentFlags().String("target", "", "")
	rootCmd.PersistentFlags().String("environment", "", "")
	feedbackCmd := &cobra.Command{Use: "feedback"}
	rootCmd.AddCommand(feedbackCmd)
	for _, path := range []string{"air run", "bundle deploy", "bundle deployment bind", "version"} {
		parent := rootCmd
		for name := range strings.FieldsSeq(path) {
			var child *cobra.Command
			for _, c := range parent.Commands() {
				if c.Name() == name {
					child = c
				}
			}
			if child == nil {
				child = &cobra.Command{Use: name}
				parent.AddCommand(child)
			}
			parent = child
		}
	}
	return feedbackCmd
}

func TestCheckArgs(t *testing.T) {
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr string
	}{
		{name: "one message", args: []string{"deploy is slow"}},
		{name: "no message", wantErr: "accepts 1 arg(s), received 0"},
		{
			name:    "unquoted message",
			args:    []string{"costs", "$HOME", "or", "`whoami`"},
			wantErr: "the message must be one argument, but received 4; put quotes around the whole message",
		},
		{
			name:    "unquoted --command",
			command: "bundle",
			args:    []string{"deploy", "deploy is slow"},
			wantErr: `--command takes the command as one argument; put quotes around "bundle deploy"`,
		},
		{
			name:    "unquoted nested --command",
			command: "bundle",
			args:    []string{"deployment", "bind", "bind fails"},
			wantErr: `--command takes the command as one argument; put quotes around "bundle deployment bind"`,
		},
		{
			name:    "words that are not a subcommand",
			command: "bundle",
			args:    []string{"nope", "slow"},
			wantErr: "the message must be one argument, but received 2; put quotes around the whole message",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkArgs(newTestFeedbackCommand(), tt.command, tt.args)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestNotSent(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nothing accepted", err: fmt.Errorf("wrapped: %w", &telemetry.NotAcceptedError{Accepted: 0, Sent: 1}), want: true},
		{name: "rejected", err: &apierr.APIError{StatusCode: 403}, want: true},
		{name: "server error", err: &apierr.APIError{StatusCode: 500}},
		{name: "connection reset", err: errors.New("connection reset by peer")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, notSent(tt.err))
		})
	}
}

func TestWorkspaceLabel(t *testing.T) {
	tests := []struct {
		name        string
		workspaceID string
		want        string
	}{
		{name: "workspace host", want: "https://workspace.test"},
		{name: "unified host", workspaceID: "123", want: "https://workspace.test (workspace ID 123)"},
		{name: "no workspace sentinel", workspaceID: "none", want: "https://workspace.test"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, workspaceLabel(&config.Config{Host: "https://workspace.test", WorkspaceID: tt.workspaceID}))
		})
	}
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
		{name: "zero run ID", opts: options{runID: "0"}, message: "x", wantErr: runIDError},
		{name: "non-numeric run ID", opts: options{runID: "abc"}, message: "x", wantErr: runIDError},
		{name: "blank message", message: " ", wantErr: messageError},
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
			want: "  line one\n  \tline two\n\nWorkspace: https://workspace.test\nDatabricks may contact you: no\n",
		},
		{
			name: "all fields",
			s:    submission{text: "slow", command: "air run", runID: "42", canBeContacted: true},
			want: "  slow\n\nWorkspace: https://workspace.test\nCommand: air run\nRun ID: 42\nDatabricks may contact you: yes\n",
		},
		{
			name: "message lines cannot pass for fields",
			s:    submission{text: "slow\nDatabricks may contact you: no", canBeContacted: true},
			want: "  slow\n  Databricks may contact you: no\n\nWorkspace: https://workspace.test\nDatabricks may contact you: yes\n",
		},
		{
			name: "control characters cannot hide text",
			s:    submission{text: "great\x1b[8m hidden\x1b[0m\r\u0085"},
			want: `  great\x1b[8m hidden\x1b[0m\r\u0085` + "\n\nWorkspace: https://workspace.test\nDatabricks may contact you: no\n",
		},
		{
			name: "format characters cannot reorder or hide text",
			s:    submission{text: "a\u202eb\u200bc\U000E0041 d\u00a0é"},
			want: `  a\u202eb\u200bc\U000e0041 d` + "\u00a0é\n\nWorkspace: https://workspace.test\nDatabricks may contact you: no\n",
		},
		{
			name: "variation selectors and fillers cannot carry hidden text",
			s:    submission{text: "ok👍\ufe0f\U000E0100\u3164\u034f e\u0301"},
			want: `  ok👍\ufe0f\U000e0100\u3164\u034f ` + "e\u0301\n\nWorkspace: https://workspace.test\nDatabricks may contact you: no\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, summary(tt.s, testWorkspace))
		})
	}
}

func TestConfirm(t *testing.T) {
	t.Run("accepted", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("y\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, submission{text: "exact feedback"}, testWorkspace)
		require.NoError(t, err)
		assert.True(t, confirmed)
		assert.Contains(t, stderr.String(), "The following feedback will be sent to Databricks:\n\n  exact feedback\n\nWorkspace: https://workspace.test\nDatabricks may contact you: no\n\n")
		assert.Contains(t, stderr.String(), "Send this feedback? [y/N]: ")
	})

	t.Run("declined", func(t *testing.T) {
		var stderr bytes.Buffer
		ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader("\n"), io.Discard, &stderr))
		confirmed, err := confirm(ctx, submission{text: "exact feedback"}, testWorkspace)
		require.NoError(t, err)
		assert.False(t, confirmed)
		assert.Contains(t, stderr.String(), "Feedback not sent.")
	})
}

func TestCheckApproval(t *testing.T) {
	const refusal = "feedback from an AI agent must be confirmed by the user; run with --yes once the user approves the message\n\n" +
		"Note for AI agents (test-agent): show the user the exact feedback message and pass --yes only after the user approves sending it"
	tests := []struct {
		name     string
		agent    string
		yes      bool
		terminal bool
		envKey   string
		envValue string
		wantErr  string
	}{
		{name: "human without a terminal"},
		{name: "agent with --yes", agent: "test-agent", yes: true},
		{name: "agent with a terminal", agent: "test-agent", terminal: true},
		{name: "agent with a terminal and NO_COLOR", agent: "test-agent", terminal: true, envKey: "NO_COLOR", envValue: "1"},
		{name: "agent with a dumb terminal", agent: "test-agent", terminal: true, envKey: "TERM", envValue: "dumb"},
		{name: "agent without a terminal", agent: "test-agent", wantErr: refusal},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testutil.SetAgent(t, tt.agent)
			ctx := t.Context()
			if tt.envKey != "" {
				ctx = env.Set(ctx, tt.envKey, tt.envValue)
			}
			var in io.Reader = strings.NewReader("")
			if tt.terminal {
				in = cmdio.FakeTTYReader(in)
			}
			ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, in, io.Discard, cmdio.FakeTTY(io.Discard), "", ""))
			err := checkApproval(ctx, tt.yes)
			if tt.wantErr != "" {
				assert.EqualError(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
		})
	}
}

func TestSendAsksBeforeSending(t *testing.T) {
	tests := []struct {
		name     string
		answer   string
		wantSent []string
		wantErr  error
	}{
		{name: "approved", answer: "y\n", wantSent: []string{"exact feedback"}},
		{name: "declined", answer: "n\n", wantErr: root.ErrAlreadyPrinted},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start the server and the client before SetAgent clears the
			// environment: on Windows, sockets need SystemRoot.
			server := testserver.New(t)
			var sent []string
			server.Handle(http.MethodPost, "/telemetry-ext", func(req testserver.Request) any {
				// The handler runs on the server's goroutine, so it must not
				// call require.
				var body telemetry.RequestBody
				assert.NoError(t, json.Unmarshal(req.Body, &body))
				for _, raw := range body.ProtoLogs {
					var entry protos.FrontendLog
					if assert.NoError(t, json.Unmarshal([]byte(raw), &entry)) {
						sent = append(sent, entry.Entry.FastFeedbackLog.Text)
					}
				}
				return telemetry.ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
			})
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
			require.NoError(t, err)
			testutil.SetAgent(t, "test-agent")

			var stdout, stderr bytes.Buffer
			ctx := cmdio.InContext(t.Context(), cmdio.NewTestIO(strings.NewReader(tt.answer), &stdout, &stderr))
			ctx = cmdctx.SetWorkspaceClient(ctx, w)
			ctx = cmdctx.GenerateExecId(ctx)
			err = send(ctx, submission{text: "exact feedback", component: defaultComponent, origin: "cli"}, false)

			assert.ErrorIs(t, err, tt.wantErr)
			assert.Equal(t, tt.wantSent, sent)
			assert.Contains(t, stderr.String(), "  exact feedback\n")
			if tt.wantErr == nil {
				assert.Contains(t, stdout.String(), "Thank you! Your feedback was sent.")
			} else {
				assert.Empty(t, stdout.String())
			}
		})
	}
}
