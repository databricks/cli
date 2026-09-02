package aitools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// Package-level seams for testability. Tests override these via helpers in
// install_test.go.
var (
	promptAgentSelection     = defaultPromptAgentSelection
	promptProceed            = defaultPromptProceed
	installSkillsForAgentsFn = installer.InstallSkillsForAgents
	installPluginForAgentFn  = installer.InstallPluginForAgent
	recordPluginInstallsFn   = installer.RecordPluginInstalls
	cleanupLegacyFn          = installer.RemoveLegacyRawSkills
)

// delivery is how the databricks tools are delivered to one agent.
type delivery int

const (
	// deliveryPlugin installs the databricks plugin through the agent's own CLI.
	deliveryPlugin delivery = iota
	// deliverySkills copies raw skill files (agents with no headless plugin
	// install: OpenCode, Antigravity, Cursor; or any agent under --skills-only).
	deliverySkills
	// deliverySkip does nothing for the agent and explains why.
	deliverySkip
)

// String returns the delivery name used in `list --output json`, so the install
// plan and the list output name the same thing the same way.
func (d delivery) String() string {
	switch d {
	case deliveryPlugin:
		return "plugin"
	case deliverySkills:
		return "skills"
	case deliverySkip:
		return "skip"
	default:
		return "unknown"
	}
}

// agentPlanItem is the resolved plan for one agent: what we'll do and why.
type agentPlanItem struct {
	agent     *agents.Agent
	delivery  delivery
	scope     string                      // agent-native plugin scope (deliveryPlugin only)
	reason    string                      // why the agent is skipped (deliverySkip only)
	skipError protos.AitoolsErrorCategory // error category for the skip (deliverySkip only)
	explicit  bool                        // named via --agents (blocking it is an error)
}

// agentChoice is one row in the interactive agent picker.
type agentChoice struct {
	agent     *agents.Agent
	label     string
	preselect bool
}

func NewInstallCmd() *cobra.Command {
	var skillsFlag, agentsFlag, scopeFlag, pathFlag string
	var includeExperimental, skillsOnly bool
	var projectFlag, globalFlag bool

	cmd := &cobra.Command{
		Use: "install",
		// Resolve auth best-effort so telemetry can upload; see tryConfigureAuth.
		PreRunE: tryConfigureAuth,
		Short:   "Install Databricks skills and plugins for coding agents",
		Long: `Install Databricks skills and plugins for detected coding agents.

By default this installs the databricks plugin through each agent's own CLI
(Claude Code, Codex, GitHub Copilot). Agents without a headless plugin install
(` + strings.Join(agents.SkillsOnlyNames(), ", ") + `) get raw skill files.

Escape hatches:
  --skills-only          Force raw skill files for every agent (no plugin).
  --skills name1,name2   Install only the named skills (with --skills-only/--path).
  --path <dir>           Write resolved skill files to a directory (no agents, no state).

Agent selection:
  --agents <name>[,...]  Act only on the named agents (works for undetected ones).
  (unset, interactive)   A picker over all known agents, detected ones pre-checked.
  (unset, non-interactive) Act on every detected agent.

Supported agents: ` + strings.Join(agents.SupportedNames(), ", "),
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)

			if skillsOnly && pathFlag != "" {
				return errors.New("cannot use --skills-only with --path; --path always writes raw skill files")
			}

			opts := installer.InstallOptions{
				IncludeExperimental: includeExperimental,
				SpecificSkills:      splitAndTrim(skillsFlag),
			}

			// --skills cherry-picks individual skill files, which only applies to
			// raw-skills delivery. The plugin is installed in full, so reject
			// --skills unless raw skills were requested via --skills-only or --path.
			if len(opts.SpecificSkills) > 0 && !skillsOnly && pathFlag == "" {
				return errors.New("--skills requires --skills-only or --path; the databricks plugin is installed in full")
			}

			src := &installer.GitHubManifestSource{}

			// --path is a dumb dump: no agents, no scope, no state.
			if pathFlag != "" {
				_, err := installer.DumpSkillsToPath(ctx, src, pathFlag, opts)
				return err
			}

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, false)
			if err != nil {
				return err
			}

			// JSON output must be fully non-interactive: every choice has to come
			// from flags so no scope prompt, agent picker, or confirm is shown.
			// Require the flags those prompts would otherwise resolve, and fail
			// fast naming them.
			if jsonMode {
				var missing []string
				if !projectFlag && !globalFlag {
					missing = append(missing, "--scope")
				}
				if agentsFlag == "" {
					missing = append(missing, "--agents")
				}
				if len(missing) > 0 {
					return fmt.Errorf("--output json requires %s so the command runs without interactive prompts", strings.Join(missing, " and "))
				}
			}

			scope, err := resolveScopeWithPrompt(ctx, projectFlag, globalFlag)
			if err != nil {
				return err
			}
			opts.Scope = scope

			// Resolve the agents to act on.
			var targetAgents []*agents.Agent
			explicit := agentsFlag != ""
			if explicit {
				targetAgents, err = resolveAgentNames(ctx, agentsFlag)
				if err != nil {
					return err
				}
			} else {
				targetAgents, err = selectAgents(ctx, scope, skillsOnly)
				if err != nil {
					return err
				}
				if len(targetAgents) == 0 {
					if jsonMode {
						return renderInstallJSON(cmd.OutOrStdout(), installOutput{Scope: scope, Agents: []agentResultJSON{}})
					}
					printNoAgentsMessage(ctx)
					return nil
				}
			}

			plan := buildPlan(targetAgents, scope, skillsOnly, explicit)

			// In the interactive picker path, show a plan summary and confirm.
			if !explicit && cmdio.IsPromptSupported(ctx) {
				printPlanSummary(ctx, plan, scope)
				proceed, err := promptProceed()
				if err != nil {
					return err
				}
				if !proceed {
					cmdio.LogString(ctx, "Cancelled.")
					return nil
				}
			}

			var outcomes []agentOutcome
			var runErr error
			defer func() {
				logInstallEvent(ctx, plan, installOpts{
					Scope:        opts.Scope,
					Experimental: opts.IncludeExperimental,
				}, classifyInstallError(runErr), outcomes)
			}()

			outcomes, runErr = executePlan(ctx, src, plan, opts, jsonMode)
			if jsonMode {
				if jerr := renderInstallJSON(cmd.OutOrStdout(), buildInstallOutput(opts.Scope, outcomes)); jerr != nil {
					return jerr
				}
				// The rendered JSON already carries the per-agent error categories,
				// so don't also print runErr as a text "Error:" line. Silence
				// cobra's error/usage output; the non-zero exit still comes from
				// returning runErr.
				cmd.SilenceErrors = true
				cmd.SilenceUsage = true
			}
			return runErr
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to install (comma-separated)")
	cmd.Flags().StringVar(&agentsFlag, "agents", "", "Agents to install for (comma-separated, e.g. claude-code,cursor)")
	cmd.Flags().BoolVar(&includeExperimental, "experimental", false, "Include experimental skills")
	cmd.Flags().BoolVar(&skillsOnly, "skills-only", false, "Force raw skill files for every agent instead of the plugin")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Write resolved skill files to this directory (no agents, no state)")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Install scope: project or global (default: global, or prompt when interactive)")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Install to project directory (cwd)")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Install globally (default)")
	markScopeBoolsDeprecated(cmd)
	return cmd
}

func installOutputIsJSON(cmd *cobra.Command) bool {
	f := cmd.Flag("output")
	if f == nil {
		return false
	}
	out, ok := f.Value.(*flags.Output)
	return ok && *out == flags.OutputJSON
}

// selectAgents returns the agents to act on when --agents is not given. The
// interactive path shows a picker over all known agents; the non-interactive
// path acts on detected agents, matching today's default. Skills delivery only
// needs a config dir, so in --skills-only mode an agent is "detected" by its
// config dir (PATH-independent); plugin delivery additionally detects agents by
// their CLI binary on PATH, which fixes the Codex/Copilot config-dir miss.
func selectAgents(ctx context.Context, scope string, skillsOnly bool) ([]*agents.Agent, error) {
	// Interactive: the picker decides; a prompt error or empty selection is a real
	// error, not a "nothing detected" no-op.
	if cmdio.IsPromptSupported(ctx) {
		choices := agentChoices(ctx, scope, skillsOnly)
		if len(choices) == 0 {
			// Agents were detected but none can be acted on in this scope; the
			// caller prints the no-agents message rather than showing an empty picker.
			return nil, nil
		}
		return promptAgentSelection(ctx, choices)
	}

	var selected []*agents.Agent
	for _, a := range agents.Registry {
		detected := a.Detected(ctx)
		if !skillsOnly {
			detected = detected || a.HasBinary(ctx)
		}
		if detected {
			selected = append(selected, a)
		}
	}
	return selected, nil
}

// agentChoices builds the interactive picker rows. Every detected agent is shown
// in the detection list with its state, but only agents that can actually be
// acted on in the chosen scope (plugin or skills delivery) become selectable
// options. Agents that would be skipped (e.g. a files-only agent under project
// scope) are listed with their reason but are not checkboxes, so the picker never
// offers an option that does nothing.
func agentChoices(ctx context.Context, scope string, skillsOnly bool) []agentChoice {
	cmdio.LogString(ctx, "Detecting coding agents...")
	var choices []agentChoice
	for _, a := range agents.Registry {
		item := planItemFor(a, scope, skillsOnly, false)
		label := agentChoiceLabel(ctx, a, item)
		cmdio.LogString(ctx, fmt.Sprintf("  %-16s %s", a.DisplayName, label))
		if item.delivery == deliverySkip {
			continue
		}
		choices = append(choices, agentChoice{
			agent:     a,
			label:     a.DisplayName + "  " + label,
			preselect: a.IsPreselected(ctx),
		})
	}
	return choices
}

// agentChoiceLabel is the picker label: the detection state, plus the skip
// reason when the agent can't be acted on in the chosen scope.
func agentChoiceLabel(ctx context.Context, a *agents.Agent, item agentPlanItem) string {
	label := agentStateLabel(a.DisplayState(ctx))
	if item.delivery == deliverySkip {
		return label + " · " + item.reason
	}
	return label
}

// agentStateLabel is the short human label for a detection state.
func agentStateLabel(s agents.DisplayState) string {
	switch s {
	case agents.StateAvailable:
		return "plugin"
	case agents.StateInstalledCLIMissing:
		return "plugin · CLI not found"
	case agents.StateFilesOnly:
		return "skills"
	default:
		return "not found"
	}
}

func defaultPromptProceed() (bool, error) {
	proceed := true
	err := huh.NewConfirm().
		Title("Proceed?").
		Value(&proceed).
		Run()
	if err != nil {
		return false, err
	}
	return proceed, nil
}

func defaultPromptAgentSelection(_ context.Context, choices []agentChoice) ([]*agents.Agent, error) {
	options := make([]huh.Option[string], 0, len(choices))
	byName := make(map[string]*agents.Agent, len(choices))
	for _, c := range choices {
		options = append(options, huh.NewOption(c.label, c.agent.Name).Selected(c.preselect))
		byName[c.agent.Name] = c.agent
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title("Select agents to set up").
		Description("space to toggle, enter to confirm").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, err
	}
	if len(selected) == 0 {
		return nil, errors.New("at least one agent must be selected")
	}

	result := make([]*agents.Agent, 0, len(selected))
	for _, name := range selected {
		result = append(result, byName[name])
	}
	return result, nil
}

// buildPlan resolves the per-agent delivery and scope. Plugin-first: an agent
// with a headless plugin install gets the plugin; --skills-only forces skills
// everywhere; agents with no plugin always get skills.
func buildPlan(targetAgents []*agents.Agent, scope string, skillsOnly, explicit bool) []agentPlanItem {
	plan := make([]agentPlanItem, 0, len(targetAgents))
	for _, a := range targetAgents {
		plan = append(plan, planItemFor(a, scope, skillsOnly, explicit))
	}
	return plan
}

// planItemFor resolves the delivery and scope for a single agent in the given
// install scope. It is shared by buildPlan and the interactive picker so the
// picker and the plan agree on what an agent will (or won't) do.
func planItemFor(a *agents.Agent, scope string, skillsOnly, explicit bool) agentPlanItem {
	item := agentPlanItem{agent: a, explicit: explicit}
	switch {
	case skillsOnly || a.Plugin == nil:
		// Raw-skills delivery (no-plugin agents, or --skills-only). Only some agents
		// support project-scoped skills, so skip the rest up front instead of
		// offering an option that fails at install time.
		if scope == installer.ScopeProject && !a.SupportsProjectScope {
			item.delivery = deliverySkip
			item.reason = "does not support project-scoped skills"
			item.skipError = protos.AitoolsErrorCategoryUnsupportedScope
		} else {
			item.delivery = deliverySkills
		}
	default:
		nativeScope, ok, reason := mapAgentScope(a, scope)
		if !ok {
			item.delivery = deliverySkip
			item.reason = reason
			item.skipError = protos.AitoolsErrorCategoryUnsupportedScope
		} else {
			item.delivery = deliveryPlugin
			item.scope = nativeScope
		}
	}
	return item
}

// printPlanSummary renders the interactive plan summary before the confirm.
func printPlanSummary(ctx context.Context, plan []agentPlanItem, scope string) {
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Plan ("+scope+" scope):")
	for _, it := range plan {
		switch it.delivery {
		case deliveryPlugin:
			cmdio.LogString(ctx, "  "+it.agent.DisplayName+"  install the databricks plugin")
		case deliverySkills:
			cmdio.LogString(ctx, "  "+it.agent.DisplayName+"  install skills")
		case deliverySkip:
			cmdio.LogString(ctx, "  "+it.agent.DisplayName+"  skip ("+it.reason+")")
		}
	}
	cmdio.LogString(ctx, "")
}

// agentOutcome is one agent's result after executePlan: how the databricks
// tools were delivered (or attempted), and, when the agent did not succeed, the
// failure category and a human-readable message for --output json. The category
// is what telemetry records; the message is local-only and never sent.
type agentOutcome struct {
	agent         *agents.Agent
	delivery      delivery
	status        outcomeStatus
	errorCategory protos.AitoolsErrorCategory // Unspecified when status == outcomeInstalled
	message       string                      // set when skipped or failed
}

type outcomeStatus string

const (
	outcomeInstalled outcomeStatus = "installed"
	outcomeSkipped   outcomeStatus = "skipped"
	outcomeFailed    outcomeStatus = "failed"
)

// executePlan carries out the plan and returns each agent's outcome. Skills
// installs go through the existing skills path. Plugin installs are reported but
// never silently fall back to skills: a blocked install is a warning (exit 0),
// unless the agent was explicitly named via --agents, which is an error.
func executePlan(ctx context.Context, src installer.ManifestSource, plan []agentPlanItem, opts installer.InstallOptions, quiet bool) ([]agentOutcome, error) {
	var skillsAgents []*agents.Agent
	var pluginItems, skipItems []agentPlanItem
	for _, it := range plan {
		switch it.delivery {
		case deliverySkills:
			skillsAgents = append(skillsAgents, it.agent)
		case deliveryPlugin:
			pluginItems = append(pluginItems, it)
		case deliverySkip:
			skipItems = append(skipItems, it)
		}
	}

	var outcomes []agentOutcome
	var explicitErrs []error

	if len(skillsAgents) > 0 {
		if !quiet {
			installer.PrintInstallingFor(ctx, skillsAgents)
		}
		// A skills install runs as a group; on failure the whole command fails and
		// the top-level error category classifies it.
		if err := installSkillsForAgentsFn(ctx, src, skillsAgents, opts); err != nil {
			return outcomes, err
		}
		for _, a := range skillsAgents {
			outcomes = append(outcomes, agentOutcome{agent: a, delivery: deliverySkills, status: outcomeInstalled})
		}
	}

	pluginCount := 0
	if len(pluginItems) > 0 {
		ref, _, err := installer.GetSkillsRef(ctx)
		if err != nil {
			return outcomes, err
		}
		records := map[string]installer.PluginRecord{}
		for _, it := range pluginItems {
			if !quiet {
				cmdio.LogString(ctx, fmt.Sprintf("Installing databricks plugin for %s...", it.agent.DisplayName))
			}
			rec, err := installPluginForAgentFn(ctx, it.agent, it.scope, ref)
			if err != nil {
				if !quiet {
					cmdio.LogString(ctx, cmdio.Yellow(ctx, fmt.Sprintf("Skipped %s: %v", it.agent.DisplayName, err)))
				}
				outcomes = append(outcomes, agentOutcome{
					agent:         it.agent,
					delivery:      deliveryPlugin,
					status:        outcomeFailed,
					errorCategory: classifyInstallError(err),
					message:       err.Error(),
				})
				if it.explicit {
					explicitErrs = append(explicitErrs, err)
				}
				continue
			}
			records[it.agent.Name] = rec
			pluginCount++
			outcomes = append(outcomes, agentOutcome{agent: it.agent, delivery: deliveryPlugin, status: outcomeInstalled})
			// Remove any raw skills we previously dropped on this agent so the
			// plugin and leftover files don't surface the same skills twice.
			if err := cleanupLegacyFn(ctx, it.agent, opts.Scope); err != nil {
				log.Debugf(ctx, "Legacy skill cleanup for %s failed: %v", it.agent.DisplayName, err)
			}
			if !quiet {
				cmdio.LogString(ctx, fmt.Sprintf("  %s  databricks plugin %s", it.agent.DisplayName, versionToken(rec.Version)))
			}
		}
		if len(records) > 0 {
			if err := recordPluginInstallsFn(ctx, opts.Scope, records, ref); err != nil {
				return outcomes, err
			}
		}
	}

	for _, it := range skipItems {
		if !quiet {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, "Skipped "+it.agent.DisplayName+": "+it.reason))
		}
		outcomes = append(outcomes, agentOutcome{
			agent:         it.agent,
			delivery:      deliverySkip,
			status:        outcomeSkipped,
			errorCategory: it.skipError,
			message:       it.reason,
		})
		if it.explicit {
			explicitErrs = append(explicitErrs, fmt.Errorf("%s: %s", it.agent.DisplayName, it.reason))
		}
	}

	if pluginCount > 0 && !quiet {
		noun := "agent"
		if pluginCount != 1 {
			noun = "agents"
		}
		cmdio.LogString(ctx, fmt.Sprintf("Installed the plugin for %d %s.", pluginCount, noun))
	}

	if len(explicitErrs) > 0 {
		return outcomes, errors.Join(explicitErrs...)
	}
	return outcomes, nil
}

type installOutput struct {
	Scope  string            `json:"scope"`
	Agents []agentResultJSON `json:"agents"`
}

type agentResultJSON struct {
	Name          string `json:"name"`
	Delivery      string `json:"delivery"`
	Status        string `json:"status"`
	ErrorCategory string `json:"errorCategory,omitempty"`
	Message       string `json:"message,omitempty"`
}

func buildInstallOutput(scope string, outcomes []agentOutcome) installOutput {
	out := installOutput{Scope: scope, Agents: make([]agentResultJSON, 0, len(outcomes))}
	for _, o := range outcomes {
		entry := agentResultJSON{
			Name:     o.agent.Name,
			Delivery: o.delivery.String(),
			Status:   string(o.status),
			Message:  o.message,
		}
		if o.errorCategory != protos.AitoolsErrorCategoryUnspecified {
			entry.ErrorCategory = string(o.errorCategory)
		}
		out.Agents = append(out.Agents, entry)
	}
	return out
}

func renderInstallJSON(w io.Writer, out installOutput) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// resolveAgentNames parses a comma-separated list of agent names and validates
// them against the registry. Returns an error for unrecognized names.
func resolveAgentNames(_ context.Context, names string) ([]*agents.Agent, error) {
	available := make(map[string]*agents.Agent, len(agents.Registry))
	var availableNames []string
	for _, a := range agents.Registry {
		available[a.Name] = a
		availableNames = append(availableNames, a.Name)
	}

	var result []*agents.Agent
	seen := make(map[string]bool)
	for name := range strings.SplitSeq(names, ",") {
		name = strings.TrimSpace(name)
		if name == "" || seen[name] {
			continue
		}
		seen[name] = true
		agent, ok := available[name]
		if !ok {
			return nil, fmt.Errorf("unknown agent %q. Available agents: %s", name, strings.Join(availableNames, ", "))
		}
		result = append(result, agent)
	}

	if len(result) == 0 {
		return nil, errors.New("no agents specified")
	}
	return result, nil
}

// printNoAgentsMessage prints the "no agents detected" message.
func printNoAgentsMessage(ctx context.Context) {
	cmdio.LogString(ctx, cmdio.Yellow(ctx, "No supported coding agents found on PATH."))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported: "+strings.Join(agents.SupportedNames(), ", ")+".")
	cmdio.LogString(ctx, "Install one, then re-run 'databricks aitools install'.")
}
