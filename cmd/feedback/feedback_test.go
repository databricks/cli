package feedback

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/databricks/cli/internal/testutil"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/cli/libs/testserver"
	"github.com/databricks/databricks-sdk-go"
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

// newTestFeedbackCommand returns the feedback command under a root with the
// root's bundle target flags.
func newTestFeedbackCommand() *cobra.Command {
	rootCmd := &cobra.Command{Use: "databricks"}
	rootCmd.PersistentFlags().String("target", "", "")
	rootCmd.PersistentFlags().String("environment", "", "")
	feedbackCmd := &cobra.Command{Use: "feedback"}
	rootCmd.AddCommand(feedbackCmd)
	return feedbackCmd
}

func TestCheckArgs(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "one message", args: []string{"deploy is slow"}},
		{name: "no message", wantErr: "accepts 1 arg(s), received 0"},
		{
			name:    "unquoted message or --command",
			args:    []string{"deploy", "is", "slow"},
			wantErr: "the message must be one argument, but received 3; put quotes around the whole message, and around --command if it has several words",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := checkArgs(newTestFeedbackCommand(), tt.args)
			if tt.wantErr == "" {
				assert.NoError(t, err)
				return
			}
			assert.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestResolveCommand(t *testing.T) {
	tests := []struct {
		name          string
		command       string
		wantComponent string
		wantOrigin    string
		wantErr       bool
	}{
		{name: "general", wantComponent: defaultComponent, wantOrigin: "cli"},
		{name: "blank", command: " ", wantComponent: defaultComponent, wantOrigin: "cli"},
		{name: "air subcommand", command: "air run", wantComponent: "ModelTraining.AIRuntime", wantOrigin: "air_run"},
		{name: "bundle group", command: "bundle", wantComponent: "DeveloperEcosystem.CoreDabs", wantOrigin: "bundle"},
		{name: "bundle subcommand", command: "bundle  deploy", wantComponent: "DeveloperEcosystem.CoreDabs", wantOrigin: "bundle_deploy"},
		{name: "unchecked subcommand", command: "bundle not-a-command", wantComponent: "DeveloperEcosystem.CoreDabs", wantOrigin: "bundle_not-a-command"},
		{name: "other command", command: "jobs list", wantComponent: defaultComponent, wantOrigin: "jobs_list"},
		{name: "unchecked top level", command: "bundel deploy", wantComponent: defaultComponent, wantOrigin: "bundel_deploy"},
		{name: "flag", command: "bundle deploy -t dev", wantErr: true},
		{name: "path argument", command: "bundle sync ./src", wantErr: true},
		{name: "uppercase", command: "Bundle deploy", wantErr: true},
		{name: "control characters", command: "bundle \x1b[8mdeploy", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			component, origin, err := resolveCommand(tt.command)
			if tt.wantErr {
				assert.EqualError(t, err, fmt.Sprintf(`invalid --command %q; it takes a command path such as "bundle deploy", without the databricks prefix, flags or arguments`, tt.command))
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
				text: "slow", component: "DeveloperEcosystem.CoreDabs",
				origin: "bundle_deploy", runID: "123", canBeContacted: true,
			},
		},
		{name: "zero run ID", opts: options{runID: "0"}, message: "x", wantErr: runIDError},
		{name: "non-numeric run ID", opts: options{runID: "abc"}, message: "x", wantErr: runIDError},
		{name: "blank message", message: " ", wantErr: messageError},
		{
			name: "long command", opts: options{command: "bundle " + strings.Repeat("a", commandLimit)}, message: "x",
			wantErr: fmt.Sprintf("--command must be at most %d characters", commandLimit),
		},
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

func TestSendWithoutConfirmation(t *testing.T) {
	tests := []struct {
		name     string
		agent    string
		terminal bool
	}{
		{name: "human without a terminal"},
		{name: "agent without a terminal", agent: "test-agent"},
		{name: "agent with a terminal", agent: "test-agent", terminal: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Start the server and the client before SetAgent clears the
			// environment: on Windows, sockets need SystemRoot.
			server := testserver.New(t)
			var sent []protos.FastFeedbackLog
			server.Handle(http.MethodPost, "/telemetry-ext", func(req testserver.Request) any {
				// The handler runs on the server's goroutine, so it must not
				// call require.
				var body telemetry.RequestBody
				assert.NoError(t, json.Unmarshal(req.Body, &body))
				for _, raw := range body.ProtoLogs {
					var entry protos.FrontendLog
					if assert.NoError(t, json.Unmarshal([]byte(raw), &entry)) {
						sent = append(sent, *entry.Entry.FastFeedbackLog)
					}
				}
				return telemetry.ResponseBody{NumProtoSuccess: int64(len(body.ProtoLogs))}
			})
			w, err := databricks.NewWorkspaceClient(&databricks.Config{Host: server.URL, Token: "token"})
			require.NoError(t, err)
			testutil.SetAgent(t, tt.agent)

			var stdout, stderr bytes.Buffer
			var in io.Reader = strings.NewReader("")
			var out io.Writer = &stdout
			var errOut io.Writer = &stderr
			if tt.terminal {
				in = cmdio.FakeTTYReader(in)
				out = cmdio.FakeTTY(out)
				errOut = cmdio.FakeTTY(errOut)
			}
			ctx := t.Context()
			ctx = cmdio.InContext(ctx, cmdio.NewIO(ctx, flags.OutputText, in, out, errOut, "", ""))
			ctx = cmdctx.SetWorkspaceClient(ctx, w)
			ctx = cmdctx.GenerateExecId(ctx)
			err = send(ctx, submission{text: "exact feedback", component: defaultComponent, origin: "cli"})

			require.NoError(t, err)
			require.Len(t, sent, 1)
			assert.Equal(t, "exact feedback", sent[0].Text)
			assert.False(t, sent[0].CanBeContacted)
			if tt.agent != "" {
				assert.Contains(t, sent[0].UserAgent, "agent/"+tt.agent)
			}
			assert.Empty(t, stderr.String())
			assert.Contains(t, stdout.String(), "Thank you! Your feedback was sent. Reference: ")
		})
	}
}
