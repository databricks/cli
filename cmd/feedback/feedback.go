package feedback

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
	"uuid"

	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/internal/build"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	messageLimit = 10000
	// commandLimit is well above the longest command path, about 80 characters.
	commandLimit = 200
	sendTimeout  = 30 * time.Second

	defaultComponent = "DeveloperEcosystem.DatabricksCLI"
	runIDKey         = "run-id"

	legacyConfirmationFlag        = "yes"
	legacyConfirmationDeprecation = "omit this flag when sending feedback; skip the feedback command when declining"
)

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

type options struct {
	command        string
	runID          string
	canBeContacted bool
	yes            bool
}

// submission is the validated input, checked before authenticating so that a
// mistake is reported without a login prompt or a network call.
type submission struct {
	text           string
	component      string
	origin         string
	runID          string
	canBeContacted bool
}

// Spec declares the feedback command.
var Spec = root.CommandSpec{
	Use:     "feedback MESSAGE",
	Short:   "Send feedback about the Databricks CLI to Databricks",
	GroupID: "development",
	Build:   buildCommand,
}

func buildCommand(ctx context.Context, cmd *cobra.Command) {
	cmd.Long = `Send feedback about the Databricks CLI to Databricks.

Describe the command you ran, what happened, what you expected, and any
workaround. Use --command to name the command the feedback is about so it
reaches the team that owns it.

The message is sent with the workspace and user you are authenticated as and
the CLI user agent, which includes the CLI version, operating system and, if
detected, the AI agent running the CLI. No logs, files or transcripts are
collected. Do not include customer content, personally identifiable information
(PII), credentials or other sensitive data. Follow-up contact is opt-in with
--can-be-contacted. Feedback is sent even when DATABRICKS_CLI_DISABLE_TELEMETRY
is set, because this command explicitly submits feedback.

AI agents can submit feedback directly without asking the user for approval.
Use --can-be-contacted only if the user explicitly agrees to be contacted.
After a successful submission, tell the user that feedback was sent and include
the reference ID.`
	cmd.Example = `  databricks feedback --command "bundle deploy" "deploy should print which resources changed"
  databricks feedback --command "air run" --run-id 123 "logs omitted the failure reason"`

	var opts options
	cmd.Flags().StringVar(&opts.command, "command", "", `CLI command the feedback is about, for example "bundle deploy"`)
	cmd.Flags().StringVar(&opts.runID, "run-id", "", "ID of the job run the feedback is about")
	cmd.Flags().BoolVar(&opts.canBeContacted, "can-be-contacted", false, "Allow Databricks to contact you about this feedback")
	cmd.Flags().BoolVarP(&opts.yes, legacyConfirmationFlag, "y", false, "Deprecated; "+legacyConfirmationDeprecation)
	cmd.Flags().MarkDeprecated(legacyConfirmationFlag, legacyConfirmationDeprecation)

	cmd.Args = checkArgs

	var s submission
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		s, err = newSubmission(cmd, args[0], &opts)
		if err != nil {
			return err
		}
		if cmd.Flags().Changed(legacyConfirmationFlag) && !opts.yes && useragent.AgentProvider() != "" {
			return errors.New("feedback was not sent because --yes=false")
		}
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return send(cmd.Context(), s)
	}
}

// checkArgs requires the message as one argument. Several arguments mean that
// the message, or a --command with several words, was not quoted. The error
// says what to quote rather than printing a command to paste, which would drop
// the user's other flags and depend on their shell's quoting.
func checkArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 1 {
		return fmt.Errorf("the message must be one argument, but received %d; put quotes around the whole message, and around --command if it has several words", len(args))
	}
	return root.ExactArgs(1)(cmd, args)
}

func newSubmission(cmd *cobra.Command, message string, opts *options) (submission, error) {
	// Feedback never loads the bundle, so a target would be ignored.
	for _, name := range []string{"target", "environment"} {
		if cmd.Flag(name).Changed {
			return submission{}, fmt.Errorf("--%s selects a bundle target, which feedback does not use; use --profile to choose the workspace", name)
		}
	}
	text, err := validateMessage(message)
	if err != nil {
		return submission{}, err
	}
	if utf8.RuneCountInString(opts.command) > commandLimit {
		return submission{}, fmt.Errorf("--command must be at most %d characters", commandLimit)
	}
	component, origin, err := resolveCommand(opts.command)
	if err != nil {
		return submission{}, err
	}
	s := submission{
		text:           text,
		component:      component,
		origin:         origin,
		canBeContacted: opts.canBeContacted,
	}
	if opts.runID != "" {
		id, err := strconv.ParseInt(opts.runID, 10, 64)
		if err != nil || id <= 0 {
			return submission{}, errors.New("--run-id must be a positive job run ID")
		}
		s.runID = strconv.FormatInt(id, 10)
	}
	return s, nil
}

func send(ctx context.Context, s submission) error {
	if s.runID != "" {
		ctx = useragent.InContext(ctx, runIDKey, s.runID)
	}
	if provider := useragent.AgentProvider(); provider != "" {
		// The SDK adds the agent to HTTP request headers, but the feedback record
		// needs it too so triage can distinguish agent reports.
		ctx = useragent.InContext(ctx, useragent.AgentKey, provider)
	}
	cfg := cmdctx.WorkspaceClient(ctx).Config

	id := uuid.New().String()
	sendCtx, cancel := context.WithTimeout(ctx, sendTimeout)
	defer cancel()
	err := telemetry.Send(sendCtx, cfg, []protos.FrontendLog{{
		FrontendLogEventID: id,
		Entry: protos.FrontendLogEntry{
			FastFeedbackLog: &protos.FastFeedbackLog{
				ReferenceEventID: cmdctx.ExecId(ctx),
				ParentUI:         s.component,
				Origin:           s.origin,
				CanBeContacted:   s.canBeContacted,
				UIVersions:       "cli:" + build.GetInfo().Version,
				UserAgent:        useragent.FromContext(ctx),
				Text:             s.text,
			},
		},
	}})
	if err != nil {
		if telemetry.NotSent(err) {
			return fmt.Errorf("feedback %s was not sent: %w", id, err)
		}
		// The upload may have been accepted before the error, so retrying can
		// file the same feedback twice.
		return fmt.Errorf("feedback %s could not be confirmed as sent: %w", id, err)
	}

	return cmdio.RenderWithTemplate(ctx, receipt{Received: true, FeedbackID: id}, "", receiptTemplate)
}

func validateMessage(message string) (string, error) {
	text := strings.TrimSpace(message)
	if utf8.RuneCountInString(text) > messageLimit || !strings.ContainsFunc(text, visible) {
		return "", fmt.Errorf("feedback message must contain 1 to %d characters, including a visible one", messageLimit)
	}
	return text, nil
}

// resolveCommand returns the component and origin for --command. The first word
// picks the owning team. The command is not looked up, which would mean
// building the command tree; its words are only limited to command-name
// characters, which keeps flags, paths and control sequences out of origin.
// The origin joins the words with underscores, as in bundle_deploy.
func resolveCommand(command string) (string, string, error) {
	words := strings.Fields(command)
	if len(words) == 0 {
		return defaultComponent, "cli", nil
	}
	for _, word := range words {
		if !isCommandWord(word) {
			return "", "", fmt.Errorf(`invalid --command %q; it takes a command path such as "bundle deploy", without the databricks prefix, flags or arguments`, command)
		}
	}
	component := defaultComponent
	if productComponent, ok := productComponents[words[0]]; ok {
		component = productComponent
	}
	return component, strings.Join(words, "_"), nil
}

// isCommandWord reports whether word can be part of a command path: lowercase
// letters, digits and dashes, not starting with a dash.
func isCommandWord(word string) bool {
	if word == "" || word[0] == '-' {
		return false
	}
	for _, r := range word {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '-' {
			return false
		}
	}
	return true
}

// invisible reports whether a terminal would not show r as written, other than
// a newline or tab: characters that are not graphic, such as escape sequences,
// bidi overrides and zero-width characters, and the variation selectors and
// fillers that Unicode marks as default ignorable, which can carry hidden text.
func invisible(r rune) bool {
	if r == '\n' || r == '\t' {
		return false
	}
	return !unicode.IsGraphic(r) || unicode.In(r, unicode.Variation_Selector, unicode.Other_Default_Ignorable_Code_Point)
}

// visible reports whether r shows up as a mark in the terminal.
func visible(r rune) bool {
	return !unicode.IsSpace(r) && !invisible(r)
}
