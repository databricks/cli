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
	"github.com/databricks/cli/libs/agent"
	"github.com/databricks/cli/libs/cmdctx"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/databrickscfg/profile"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/apierr"
	"github.com/databricks/databricks-sdk-go/config"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

const (
	messageLimit = 10000
	sendTimeout  = 30 * time.Second

	defaultComponent = "DeveloperEcosystem.DatabricksCLI"
	runIDKey         = "run-id"
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
	command        string
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
collected. Do not include credentials or sensitive data. Follow-up contact is
opt-in with --can-be-contacted. Feedback is sent even when
DATABRICKS_CLI_DISABLE_TELEMETRY is set, because you asked to send it.

AI agents: submit feedback only with the user's approval. When an agent is
detected, the command asks for confirmation, or requires --yes when no terminal
is attached. Show the user the exact message and the flags you pass with it,
such as --can-be-contacted, and pass --yes only after the user approves it.`
	cmd.Example = `  databricks feedback --command "bundle deploy" "deploy should print which resources changed"
  databricks feedback --command "air run" --run-id 123 "logs omitted the failure reason"`

	var opts options
	cmd.Flags().StringVar(&opts.command, "command", "", `CLI command the feedback is about, for example "bundle deploy"`)
	cmd.Flags().StringVar(&opts.runID, "run-id", "", "ID of the job run the feedback is about")
	cmd.Flags().BoolVar(&opts.canBeContacted, "can-be-contacted", false, "Allow Databricks to contact you about this feedback")
	cmd.Flags().BoolVarP(&opts.yes, "yes", "y", false, "Confirm sending feedback written by an AI agent")

	cmd.Args = checkArgs

	var s submission
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		var err error
		s, err = newSubmission(cmd, args[0], &opts)
		if err != nil {
			return err
		}
		if err := checkApproval(cmd.Context(), opts.yes); err != nil {
			return err
		}
		cmd.SetContext(root.SkipLoadBundle(cmd.Context()))
		return root.MustWorkspaceClient(cmd, args)
	}
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return send(cmd.Context(), s, opts.yes)
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
	component, origin, err := resolveCommand(cmd, opts.command)
	if err != nil {
		return submission{}, err
	}
	s := submission{
		text:           text,
		command:        strings.Join(strings.Fields(opts.command), " "),
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

func send(ctx context.Context, s submission, yes bool) error {
	if s.runID != "" {
		ctx = useragent.InContext(ctx, runIDKey, s.runID)
	}
	if provider := useragent.AgentProvider(); provider != "" {
		// The SDK adds the agent to HTTP request headers, but the feedback record
		// needs it too so triage can distinguish agent reports.
		ctx = useragent.InContext(ctx, useragent.AgentKey, provider)
	}
	cfg := cmdctx.WorkspaceClient(ctx).Config
	if needsApproval(yes) {
		confirmed, err := confirm(ctx, s, workspaceLabel(cfg))
		if err != nil {
			return err
		}
		if !confirmed {
			// confirm told the user; the exit code tells the agent that ran the
			// command that nothing was sent.
			return root.ErrAlreadyPrinted
		}
	}

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
		if notSent(err) {
			return fmt.Errorf("feedback %s was not sent: %w", id, err)
		}
		// The upload may have been accepted before the error, so retrying can
		// file the same feedback twice.
		return fmt.Errorf("feedback %s could not be confirmed as sent: %w", id, err)
	}

	return cmdio.RenderWithTemplate(ctx, receipt{Received: true, FeedbackID: id}, "", receiptTemplate)
}

// notSent reports whether err shows that the server stored nothing: it rejected
// the request, or did not accept the feedback.
func notSent(err error) bool {
	if _, ok := errors.AsType[*telemetry.NotAcceptedError](err); ok {
		return true
	}
	apiErr, ok := errors.AsType[*apierr.APIError](err)
	return ok && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500
}

func validateMessage(message string) (string, error) {
	text := strings.TrimSpace(message)
	if utf8.RuneCountInString(text) > messageLimit || !strings.ContainsFunc(text, visible) {
		return "", fmt.Errorf("feedback message must contain 1 to %d characters, including a visible one", messageLimit)
	}
	return text, nil
}

// resolveCommand returns the component and origin for --command. Its first word
// must be a top-level command and picks the owning team, so "bundle deploy"
// reaches bundles. The rest is not checked against the command tree, which would
// mean building every command; instead every word is limited to the characters
// of command names, so flags, paths, arguments and terminal control sequences
// cannot reach origin or the approval prompt. The origin joins the words with
// underscores, as the CLI names commands elsewhere (bundle_deploy).
func resolveCommand(cmd *cobra.Command, command string) (string, string, error) {
	words := strings.Fields(command)
	if len(words) == 0 {
		return defaultComponent, "cli", nil
	}
	// Top-level commands are registered on the root before they are built.
	top, rest, err := cmd.Root().Find(words[:1])
	valid := err == nil && len(rest) == 0 && top != cmd.Root()
	for _, word := range words {
		valid = valid && isCommandWord(word)
	}
	if !valid {
		return "", "", fmt.Errorf(`unknown command %q; --command takes a command path such as "bundle deploy", without the databricks prefix, flags or arguments`, command)
	}
	// An alias resolves to the command's name.
	words[0] = top.Name()
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

// needsApproval reports whether the user must approve the feedback before it is
// sent: an AI agent is running the CLI and did not pass --yes.
func needsApproval(yes bool) bool {
	return useragent.AgentProvider() != "" && !yes
}

// checkApproval returns an error when the user must approve the feedback but
// there is no terminal to ask in. It needs only the input, so it runs before
// authenticating.
func checkApproval(ctx context.Context, yes bool) error {
	if !needsApproval(yes) || cmdio.IsInputTTY(ctx) {
		return nil
	}
	return errors.New("feedback from an AI agent must be confirmed by the user; run with --yes once the user approves the message\n\n" +
		agent.AgentNote("show the user the exact feedback message and the flags you pass with it, such as --can-be-contacted, and pass --yes only after the user approves sending it"))
}

// confirm asks the user to approve feedback an AI agent is about to send. It
// returns false without error if the user declines.
func confirm(ctx context.Context, s submission, workspace string) (bool, error) {
	cmdio.LogString(ctx, "The following feedback will be sent to Databricks:\n\n"+summary(s, workspace))
	confirmed, err := cmdio.AskYesOrNo(ctx, "Send this feedback?")
	if err != nil {
		return false, err
	}
	if !confirmed {
		cmdio.LogString(ctx, "Feedback not sent.")
	}
	return confirmed, nil
}

// workspaceLabel names the workspace feedback is filed in. A unified host
// serves several workspaces, so there the host alone does not identify it.
func workspaceLabel(cfg *config.Config) string {
	if id := cfg.WorkspaceID; id != "" && id != profile.WorkspaceIDNone {
		return fmt.Sprintf("%s (workspace ID %s)", cfg.Host, id)
	}
	return cfg.Host
}

// summary returns the message, the workspace it is filed in and the choices
// sent with it, for the user to approve.
func summary(s submission, workspace string) string {
	var b strings.Builder
	// Indent the message so that none of its lines can pass for one of the
	// fields below it.
	b.WriteString("  " + strings.ReplaceAll(escapeInvisible(s.text), "\n", "\n  "))
	b.WriteString("\n\n")
	fmt.Fprintf(&b, "Workspace: %s\n", workspace)
	if s.command != "" {
		fmt.Fprintf(&b, "Command: %s\n", s.command)
	}
	if s.runID != "" {
		fmt.Fprintf(&b, "Run ID: %s\n", s.runID)
	}
	contact := "no"
	if s.canBeContacted {
		contact = "yes"
	}
	fmt.Fprintf(&b, "Databricks may contact you: %s\n", contact)
	return b.String()
}

// escapeInvisible escapes the runes that invisible reports, so none of them can
// hide, reorder or overwrite part of the message in the terminal.
func escapeInvisible(text string) string {
	var b strings.Builder
	for _, r := range text {
		if invisible(r) {
			quoted := strconv.QuoteRuneToASCII(r)
			b.WriteString(quoted[1 : len(quoted)-1])
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
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
