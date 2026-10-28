package aircmd_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	aircmd "github.com/databricks/cli/cmd/air"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/testserver"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func feedbackCommand(t *testing.T, args ...string) (*cobra.Command, *bytes.Buffer) {
	t.Helper()
	cmd, _, err := aircmd.New().Find([]string{"feedback"})
	require.NoError(t, err)
	output := flags.OutputJSON
	cmd.Flags().Var(&output, "output", "")
	require.NoError(t, cmd.ParseFlags(args))
	buf := new(bytes.Buffer)
	ctx := cmdio.InContext(t.Context(), cmdio.NewIO(t.Context(), output, nil, buf, buf, "", ""))
	cmd.SetContext(ctx)
	return cmd, buf
}

func TestFeedbackValidation(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "test-token")
	for _, tc := range []struct {
		name  string
		args  []string
		valid bool
	}{
		{"missing", nil, false},
		{"blank", []string{"--message", " \n\t"}, false},
		{"too long", []string{"--message", strings.Repeat("a", 10001)}, false},
		{"unicode limit", []string{"--message", strings.Repeat("界", 10000)}, true},
		{"invalid reporter", []string{"--message", "broken", "--reporter", "robot"}, false},
		{"agent on human", []string{"--message", "broken", "--agent-name", "codex"}, false},
		{"invalid run", []string{"--message", "broken", "--run-id", "0"}, false},
		{"overflow run", []string{"--message", "broken", "--run-id", "9223372036854775808"}, false},
		{"agent name limit", []string{"--message", "broken", "--reporter", "agent", "--agent-name", strings.Repeat("a", 101)}, false},
		{"human", []string{"--message", "broken"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cmd, buf := feedbackCommand(t, tc.args...)
			err := cmd.PreRunE(cmd, nil)
			if tc.valid {
				require.NoError(t, err)
				return
			}
			require.ErrorIs(t, err, root.ErrAlreadyPrinted)
			var got struct {
				Error struct {
					Code      string
					Retryable bool
				}
			}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
			assert.Equal(t, "INVALID_ARGUMENT", got.Error.Code)
			assert.False(t, got.Error.Retryable)
		})
	}
}

func TestFeedbackSubmission(t *testing.T) {
	for _, tc := range []struct {
		name     string
		response telemetry.ResponseBody
		accepted bool
	}{
		{"accepted", telemetry.ResponseBody{NumProtoSuccess: 1}, true},
		{"unacknowledged", telemetry.ResponseBody{}, false},
		{"rejected", telemetry.ResponseBody{Errors: []telemetry.LogError{{ErrorType: "INVALID_LOG"}}}, false},
		{"inconsistent", telemetry.ResponseBody{NumProtoSuccess: 1, Errors: []telemetry.LogError{{ErrorType: "INVALID_LOG"}}}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := testserver.New(t)
			t.Cleanup(server.Close)
			t.Setenv("DATABRICKS_HOST", server.URL)
			t.Setenv("DATABRICKS_TOKEN", "test-token")
			t.Setenv("DATABRICKS_WORKSPACE_ID", "123")
			var event struct {
				ID    string `json:"frontend_log_event_id"`
				Entry struct {
					Feedback map[string]any `json:"fast_feedback_log"`
				} `json:"entry"`
			}
			server.Handle("POST", "/telemetry-ext", func(req testserver.Request) any {
				assert.Equal(t, "Bearer test-token", req.Headers.Get("Authorization"))
				assert.Equal(t, "123", req.Headers.Get("X-Databricks-Workspace-Id"))
				var body telemetry.RequestBody
				require.NoError(t, json.Unmarshal(req.Body, &body))
				require.NotNil(t, body.Items)
				require.Len(t, body.ProtoLogs, 1)
				require.NoError(t, json.Unmarshal([]byte(body.ProtoLogs[0]), &event))
				return tc.response
			})
			cmd, buf := feedbackCommand(t, "--message", "  logs omitted the failure reason  ", "--reporter", "agent", "--agent-name", "codex", "--run-id", "9007199254740993")
			require.NoError(t, cmd.PreRunE(cmd, nil))
			err := cmd.RunE(cmd, nil)
			assert.NotEmpty(t, event.ID)
			assert.Equal(t, event.ID, event.Entry.Feedback["reference_event_id"])
			assert.Equal(t, "logs omitted the failure reason", event.Entry.Feedback["text"])
			assert.Equal(t, "PRODUCT_AIR", event.Entry.Feedback["product"])
			assert.Equal(t, "SOURCE_AIR_CLI", event.Entry.Feedback["feedback_source"])
			assert.Equal(t, "REPORTER_AGENT", event.Entry.Feedback["reporter_type"])
			assert.Equal(t, "codex", event.Entry.Feedback["agent_name"])
			assert.Equal(t, "9007199254740993", event.Entry.Feedback["run_id"])
			assert.Equal(t, false, event.Entry.Feedback["can_be_contacted"])
			assert.Contains(t, event.Entry.Feedback, "cli_version")
			if !tc.accepted {
				require.ErrorIs(t, err, root.ErrAlreadyPrinted)
				assert.Contains(t, buf.String(), "FEEDBACK_SUBMISSION_FAILED")
				assert.NotContains(t, buf.String(), `"received": true`)
				return
			}
			require.NoError(t, err)
			var receipt struct {
				Data struct {
					Received bool
					ID       string `json:"feedback_id"`
				}
			}
			require.NoError(t, json.Unmarshal(buf.Bytes(), &receipt))
			assert.True(t, receipt.Data.Received)
			assert.Equal(t, event.ID, receipt.Data.ID)
		})
	}
}

func TestFeedbackCancelledSubmissionDoesNotReportSuccess(t *testing.T) {
	server := testserver.New(t)
	t.Cleanup(server.Close)
	t.Setenv("DATABRICKS_HOST", server.URL)
	t.Setenv("DATABRICKS_TOKEN", "test-token")
	cmd, buf := feedbackCommand(t, "--message", "logs omitted the failure")
	require.NoError(t, cmd.PreRunE(cmd, nil))
	ctx, cancel := context.WithCancel(cmd.Context())
	cancel()
	cmd.SetContext(ctx)
	err := cmd.RunE(cmd, nil)
	require.ErrorIs(t, err, root.ErrAlreadyPrinted)
	assert.Contains(t, buf.String(), "FEEDBACK_SUBMISSION_FAILED")
	assert.NotContains(t, buf.String(), `"received": true`)
}
