package aircmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/auth"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/spf13/cobra"
)

const (
	feedbackMessageLimit   = 10000
	feedbackAgentNameLimit = 100
	feedbackTimeout        = 30 * time.Second
)

type feedbackOptions struct {
	message        string
	reporter       string
	runID          string
	agentName      string
	canBeContacted bool
}

func (o *feedbackOptions) validate() error {
	o.message = strings.TrimSpace(o.message)
	if o.message == "" || !utf8.ValidString(o.message) || utf8.RuneCountInString(o.message) > feedbackMessageLimit {
		return fmt.Errorf("--message must contain 1 to %d UTF-8 characters", feedbackMessageLimit)
	}
	if o.reporter != "human" && o.reporter != "agent" {
		return errors.New("--reporter must be human or agent")
	}
	if o.runID != "" {
		id, err := strconv.ParseInt(o.runID, 10, 64)
		if err != nil || id <= 0 {
			return errors.New("--run-id must be a positive job run ID")
		}
	}
	o.agentName = strings.TrimSpace(o.agentName)
	if o.agentName != "" && o.reporter != "agent" {
		return errors.New("--agent-name requires --reporter agent")
	}
	if !utf8.ValidString(o.agentName) || utf8.RuneCountInString(o.agentName) > feedbackAgentNameLimit {
		return fmt.Errorf("--agent-name must contain at most %d UTF-8 characters", feedbackAgentNameLimit)
	}
	return nil
}

// These bindings mirror FastFeedbackLog in universe. Feedback text is sent only
// through that explicitly submitted feedback schema, never a CLI usage event.
type fastFeedbackLog struct {
	ReferenceEventID string `json:"reference_event_id"`
	ParentUI         string `json:"parent_ui"`
	Origin           string `json:"origin"`
	Text             string `json:"text"`
	CanBeContacted   bool   `json:"can_be_contacted"`
	Product          string `json:"product"`
	FeedbackSource   string `json:"feedback_source"`
	ReporterType     string `json:"reporter_type"`
	CLIVersion       string `json:"cli_version"`
	RunID            string `json:"run_id,omitempty"`
	AgentName        string `json:"agent_name,omitempty"`
}

type feedbackEvent struct {
	FrontendLogEventID string `json:"frontend_log_event_id"`
	Entry              struct {
		FastFeedbackLog fastFeedbackLog `json:"fast_feedback_log"`
	} `json:"entry"`
}

type feedbackReceipt struct {
	Received   bool   `json:"received"`
	FeedbackID string `json:"feedback_id"`
}

func newFeedbackCommand() *cobra.Command {
	var o feedbackOptions
	cmd := &cobra.Command{
		Use:   "feedback",
		Short: "Send feedback about AI Runtime to Databricks",
		Long: `Send feedback about AI Runtime using your workspace authentication.

Include the command, observed problem, expected behavior, and any workaround
in --message. Use --reporter agent for reports written by an agent.

Sends your message, CLI version, reporter type, and optional run ID and agent
name to Fast Feedback. The service associates it with your workspace and
authenticated principal. No transcripts, source files, or logs are collected.
Do not include credentials or sensitive data. Follow-up contact is opt-in.

This is an explicit submission, independent of automatic usage telemetry.
Success confirms receipt by the collector; processing into review tools is
asynchronous.`,
		Example: `  databricks air feedback --message "air logs omitted the failure reason" --reporter agent --agent-name codex --run-id 123 -o json`,
	}
	cmd.Flags().StringVar(&o.message, "message", "", "Feedback message (1–10,000 characters)")
	cmd.Flags().StringVar(&o.reporter, "reporter", "human", "Who wrote the feedback: human or agent")
	cmd.Flags().StringVar(&o.runID, "run-id", "", "Related job run ID")
	cmd.Flags().StringVar(&o.agentName, "agent-name", "", "Agent name; requires --reporter agent")
	cmd.Flags().BoolVar(&o.canBeContacted, "can-be-contacted", false, "Allow follow-up about this feedback")
	// Cobra runs Args before root initializes command IO. Validate here so JSON
	// errors can be rendered, while still rejecting input before authentication.
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if err := cobra.NoArgs(cmd, args); err != nil {
			return renderError(cmd.Context(), cmd, "INVALID_ARGUMENT", "VALIDATION", false, err)
		}
		if err := o.validate(); err != nil {
			return renderError(cmd.Context(), cmd, "INVALID_ARGUMENT", "VALIDATION", false, err)
		}
		err := root.MustWorkspaceClient(cmd, args)
		if err == nil || errors.Is(err, root.ErrAlreadyPrinted) {
			return err
		}
		return renderError(cmd.Context(), cmd, "AUTHENTICATION_ERROR", "AUTHENTICATION", false, err)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		apiClient, err := client.New(cmdctx.WorkspaceClient(ctx).Config)
		if err != nil {
			return renderError(ctx, cmd, "AUTHENTICATION_ERROR", "AUTHENTICATION", false, err)
		}
		id := uuid.New().String()
		if err := submitFeedback(ctx, apiClient, o, id); err != nil {
			// A timeout can occur after acceptance. Do not invite automated retries
			// that would create duplicate reports under a new submission ID.
			return renderError(ctx, cmd, "FEEDBACK_SUBMISSION_FAILED", "SUBMISSION", false,
				fmt.Errorf("feedback %s could not be confirmed: %w", id, err))
		}
		if root.OutputType(cmd) == flags.OutputJSON {
			return renderEnvelope(ctx, feedbackReceipt{Received: true, FeedbackID: id})
		}
		cmdio.LogString(ctx, "Feedback received. Reference: "+id)
		return nil
	}
	return cmd
}

func submitFeedback(ctx context.Context, apiClient *client.DatabricksClient, o feedbackOptions, id string) error {
	ctx, cancel := context.WithTimeout(ctx, feedbackTimeout)
	defer cancel()
	event := feedbackEvent{FrontendLogEventID: id}
	event.Entry.FastFeedbackLog = fastFeedbackLog{
		ReferenceEventID: id,
		ParentUI:         "ModelTraining.AIRuntime",
		Origin:           "air-cli",
		Text:             o.message,
		CanBeContacted:   o.canBeContacted,
		Product:          "PRODUCT_AIR",
		FeedbackSource:   "SOURCE_AIR_CLI",
		ReporterType:     "REPORTER_" + strings.ToUpper(o.reporter),
		CLIVersion:       build.GetInfo().Version,
		RunID:            o.runID,
		AgentName:        o.agentName,
	}
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("encode feedback: %w", err)
	}
	var response telemetry.ResponseBody
	err = apiClient.Do(ctx, http.MethodPost, "/telemetry-ext", auth.WorkspaceIDHeaders(apiClient.Config), nil, telemetry.RequestBody{
		UploadTime: time.Now().UnixMilli(),
		Items:      []string{},
		ProtoLogs:  []string{string(data)},
	}, &response)
	if err != nil {
		return fmt.Errorf("submit feedback: %w", err)
	}
	if len(response.Errors) != 0 || response.NumProtoSuccess != 1 {
		return errors.New("collector did not acknowledge the feedback")
	}
	return nil
}
