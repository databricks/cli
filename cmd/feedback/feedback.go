package feedback

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
	"uuid"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/client"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	messageLimit = 10000
	sendTimeout  = 30 * time.Second

	// ES component feedback is filed under unless the command it is about
	// belongs to a product listed in productComponents.
	defaultComponent = "DeveloperEcosystem.DatabricksCLI"

	// User agent key for context the fast feedback schema has no field for.
	// The CLI version and OS are already part of the user agent.
	runIDKey = "run-id"
)

// ES components for top-level commands owned by a product team, so their
// feedback reaches that team's triage queue.
var productComponents = map[string]string{
	"air":    "ModelTraining.AIRuntime",
	"bundle": "DeveloperEcosystem.CoreDabs",
}

const receiptTemplate = `Thank you! Your feedback was sent. Reference: {{.FeedbackID}}
`

type receipt struct {
	Received   bool   `json:"received"`
	FeedbackID string `json:"feedback_id"`
}

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "feedback MESSAGE",
		Short:   "Send feedback about the Databricks CLI to Databricks",
		GroupID: "development",
		Long: `Send feedback about the Databricks CLI to Databricks.

Describe the command you ran, what happened, what you expected, and any
workaround. Use --command to name the command the feedback is about so it
reaches the team that owns it.

The message is sent with the workspace and user you are authenticated as and
the CLI user agent, which includes the CLI version, operating system and, if
detected, the AI agent running the CLI. No logs, files or transcripts are
collected. Do not include credentials or sensitive data. Follow-up contact is
opt-in with --can-be-contacted.

AI agents: submit feedback only with the user's approval. When an agent is
detected, the command asks for confirmation, or requires --yes when no terminal
is attached. Show the user the exact message and pass --yes only after the user
approves it.`,
		Example: `  databricks feedback --command "bundle deploy" "deploy should print which resources changed"
  databricks feedback --command "air run" --run-id 123 "logs omitted the failure reason"`,
		Args:    root.ExactArgs(1),
		PreRunE: root.MustWorkspaceClient,
	}

	var command string
	var runID string
	var canBeContacted bool
	var yes bool
	cmd.Flags().StringVar(&command, "command", "", `CLI command the feedback is about, for example "bundle deploy"`)
	cmd.Flags().StringVar(&runID, "run-id", "", "ID of the job run the feedback is about")
	cmd.Flags().BoolVar(&canBeContacted, "can-be-contacted", false, "Allow Databricks to contact you about this feedback")
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Confirm sending feedback written by an AI agent")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()

		text := strings.TrimSpace(args[0])
		if text == "" || utf8.RuneCountInString(text) > messageLimit {
			return fmt.Errorf("feedback message must contain 1 to %d characters", messageLimit)
		}

		component := defaultComponent
		origin := "cli"
		if command != "" {
			target, rest, err := cmd.Root().Find(strings.Fields(command))
			if err != nil || len(rest) > 0 || target == cmd.Root() {
				return fmt.Errorf("unknown command %q", command)
			}
			origin = root.CommandString(target)
			if c, ok := productComponents[topLevelName(target)]; ok {
				component = c
			}
		}
		if runID != "" {
			if id, err := strconv.ParseInt(runID, 10, 64); err != nil || id <= 0 {
				return errors.New("--run-id must be a positive job run ID")
			}
			ctx = useragent.InContext(ctx, runIDKey, runID)
		}

		if provider := useragent.AgentProvider(); provider != "" {
			// The SDK adds the agent only to HTTP request headers; the feedback
			// record needs it too so triage can tell agent reports apart.
			ctx = useragent.InContext(ctx, useragent.AgentKey, provider)
			if !yes {
				ok, err := confirm(ctx, text)
				if err != nil || !ok {
					return err
				}
			}
		}

		id := uuid.New().String()
		apiClient, err := client.New(cmdctx.WorkspaceClient(ctx).Config)
		if err != nil {
			return err
		}

		sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
		defer cancel()
		err = telemetry.Send(sendCtx, apiClient, []protos.FrontendLog{{
			FrontendLogEventID: id,
			Entry: protos.FrontendLogEntry{
				FastFeedbackLog: &protos.FastFeedbackLog{
					ReferenceEventID: cmdctx.ExecId(ctx),
					ParentUI:         component,
					Origin:           origin,
					CanBeContacted:   canBeContacted,
					UIVersions:       "cli:" + build.GetInfo().Version,
					UserAgent:        useragent.FromContext(ctx),
					Text:             text,
				},
			},
		}})
		if err != nil {
			// The upload may have been accepted before the error, so retrying
			// can file the same feedback twice.
			return fmt.Errorf("feedback %s could not be confirmed as sent: %w", id, err)
		}

		return cmdio.RenderWithTemplate(ctx, receipt{Received: true, FeedbackID: id}, "", receiptTemplate)
	}

	return cmd
}

// confirm asks the user to approve feedback an AI agent is about to send. It
// returns false without error if the user declines.
func confirm(ctx context.Context, text string) (bool, error) {
	if !cmdio.IsPromptSupported(ctx) {
		return false, errors.New("feedback from an AI agent must be confirmed by the user; run with --yes once the user approves the message\n\n" +
			agent.AgentNote("show the user the exact feedback message and pass --yes only after the user approves sending it."))
	}
	cmdio.LogString(ctx, "The following feedback will be sent to Databricks:\n\n"+text+"\n")
	ok, err := cmdio.AskYesOrNo(ctx, "Send this feedback?")
	if err != nil {
		return false, err
	}
	if !ok {
		cmdio.LogString(ctx, "Feedback not sent.")
	}
	return ok, nil
}

func topLevelName(cmd *cobra.Command) string {
	for cmd.HasParent() && cmd.Parent().HasParent() {
		cmd = cmd.Parent()
	}
	return cmd.Name()
}
