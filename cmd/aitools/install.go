package aitools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/cmd/root"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/flags"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/databricks/databricks-sdk-go/useragent"
	"github.com/spf13/cobra"
)

// Package-level seams for testability. Tests override these via helpers in
// install_test.go.
var (
	promptAgentSelection     = defaultPromptAgentSelection
	installSkillsForAgentsFn = installer.InstallSkillsForAgents
	installPluginForAgentFn  = installer.InstallPluginForAgent
	recordPluginInstallsFn   = installer.RecordPluginInstalls
	cleanupLegacyFn          = installer.RemoveLegacyRawSkills
	pluginAlreadyInstalledFn = pluginAlreadyInstalled
)

// errNoAgentsSelected is returned when the picker is confirmed with nothing
// checked. The funnel reports it as CANCELLED: the user chose to act on nothing.
var errNoAgentsSelected = errors.New("at least one agent must be selected")

// installPromptsSupported reports whether install may show its prompts. Agents
// often run commands in a pseudo-terminal (Cursor, Copilot in the VS Code
// terminal) where stdin looks like a TTY but nobody can answer a TUI prompt.
// Agent detection is the SDK's list of agent environment variables:
// https://github.com/databricks/databricks-sdk-go/blob/main/useragent/agent.go
// An agent caller always gets the documented non-interactive defaults.
func installPromptsSupported(ctx context.Context) bool {
	return cmdio.IsPromptSupported(ctx) && useragent.AgentProvider() == ""
}

// agentCallerNote is printed when install skips its prompts on a terminal
// because the caller is a coding agent. It names the detected agent by its
// --agents value when it has one, and the flags of `databricks aitools install`
// that would have replaced the skipped prompts. The command is named because
// the deprecated `experimental aitools skills install` alias does not accept
// these flags.
func agentCallerNote(missingFlags []string) string {
	provider := useragent.AgentProvider()
	detected := "a coding agent was"
	if provider == "multiple" {
		detected = "more than one coding agent was"
	} else if agent := agents.AgentForProvider(provider); agent != nil {
		detected = fmt.Sprintf("a coding agent (%s) was", agent.Name)
	}
	return fmt.Sprintf("No prompts shown because %s detected. Pass %s to databricks aitools install to choose what to install.", detected, strings.Join(missingFlags, " and "))
}

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
	agent             *agents.Agent
	delivery          delivery
	scope             string                      // agent-native plugin scope (deliveryPlugin only)
	reason            string                      // why the agent is skipped (deliverySkip only)
	skipErrorCategory protos.AitoolsErrorCategory // error category for the skip (deliverySkip only)
	explicit          bool                        // named via --agents (blocking it is an error)
	// cliMissing marks a detected plugin agent whose CLI is not on PATH, so the
	// plugin cannot be installed: it gets skill files instead (deliverySkills),
	// or keeps the plugin it already has (deliveryPlugin, nothing is installed).
	cliMissing bool
	// projectPlugin marks a skills fallback in a directory where the agent
	// already loads a databricks plugin that does not cover the install scope:
	// a project or local plugin for a global install, or a personal (user or
	// local) plugin for a project install. That directory sees both.
	projectPlugin bool
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
	var auth *telemetryAuth

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install Databricks skills and plugins for coding agents",
		Long: `Install Databricks skills and plugins for detected coding agents.

By default this installs the databricks plugin through each agent's own CLI
(Claude Code, Codex, GitHub Copilot). Agents without a headless plugin install
(` + strings.Join(agents.SkillsOnlyNames(), ", ") + `) get raw skill files, and so does a
detected plugin agent whose CLI is not on PATH (for example a desktop or IDE
install) and is not named with --agents, unless it already has the plugin. For
Claude Code that is read from its own plugin list; for Codex and GitHub Copilot
only a plugin installed by 'databricks aitools' is known, so one installed
another way shows every skill twice next to the skill files. A named agent
whose CLI is not on PATH fails; add --skills-only to give it skill files.

Escape hatches:
  --skills-only          Force raw skill files for every agent (no plugin; not with --path).
  --skills name1,name2   Install only the named skills (with --skills-only/--path).
  --path <dir>           Write resolved skill files to a directory (no agents, no state).

Agent selection:
  --agents <name>[,...]  Act only on the named agents (works for undetected ones).
  (unset, interactive)   A picker over all known agents, detected ones pre-checked.
  (unset, non-interactive) Act on every detected agent.

Non-interactive runs (no terminal, or called by a coding agent) never prompt:
scope defaults to global and agents to every detected one. On a terminal, a
coding agent caller gets a note saying so.

Output:
  --output json          Emit a structured result instead of text. Requires --scope
                         and --agents so the command runs without interactive prompts.

The command exits with code 1 when it sets up nothing: no agent is detected,
every agent is skipped or fails to install, or no skill in the release can be
installed (also with --path).

Supported agents: ` + strings.Join(agents.SupportedNames(), ", "),
		Example: `  # Install a selected skill for coding agents.
  databricks aitools install --skills-only --skills databricks-ai-runtime

  # Write a selected skill to a custom directory (omit --skills-only).
  databricks aitools install --skills databricks-ai-runtime --path ./skills`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			ctx := cmd.Context()
			jsonMode := installOutputIsJSON(cmd)
			if jsonMode {
				// JSON mode emits only the structured result on stdout; silence all
				// human-readable progress (via cmdio.LogProgress) so nothing but the
				// JSON document is written.
				ctx = cmdio.WithQuiet(ctx)
			}

			interactive := installPromptsSupported(ctx)
			funnel := startFunnel(ctx, protos.AitoolsCommandInstall, installModeType(skillsOnly, pathFlag), interactive)
			defer func() { funnel.finish(ctx, err) }()

			if skillsOnly && pathFlag != "" {
				return withCategory(errors.New("cannot use --skills-only with --path; --path always writes raw skill files"), protos.AitoolsErrorCategoryInvalidFlags)
			}

			// --path is a plain file dump with no agents or install state, so there
			// is no per-agent result to report. Reject --output json here rather than
			// letting the dump run and silently emit no JSON.
			if jsonMode && pathFlag != "" {
				return withCategory(errors.New("cannot use --output json with --path; --path writes raw skill files and produces no JSON result"), protos.AitoolsErrorCategoryInvalidFlags)
			}

			opts := installer.InstallOptions{
				IncludeExperimental: includeExperimental,
				SpecificSkills:      splitAndTrim(skillsFlag),
			}

			// --skills cherry-picks individual skill files, which only applies to
			// raw-skills delivery. The plugin is installed in full, so reject
			// --skills unless raw skills were requested via --skills-only or --path.
			if len(opts.SpecificSkills) > 0 && !skillsOnly && pathFlag == "" {
				return withCategory(errors.New("--skills requires --skills-only or --path; the databricks plugin is installed in full"), protos.AitoolsErrorCategoryInvalidFlags)
			}

			src := &installer.GitHubManifestSource{}

			// --path is a dumb dump: no agents, no scope, no state.
			if pathFlag != "" {
				funnel.step(ctx, protos.AitoolsCommandStepFlagsValidated)
				// When every skill is filtered out (e.g. all experimental without
				// --experimental) this fails with ErrNoSkillsResolved, so a caller
				// does not take the empty directory for skills.
				if _, err := installer.DumpSkillsToPath(ctx, src, pathFlag, opts); err != nil {
					return withRecovery(ctx, installInvocation{path: pathFlag, skills: skillsFlag, experimental: includeExperimental}, err)
				}
				funnel.step(ctx, protos.AitoolsCommandStepSkillsDumped)
				return nil
			}

			projectFlag, globalFlag, err := parseScopeFlag(scopeFlag, projectFlag, globalFlag, false)
			if err != nil {
				return withCategory(err, protos.AitoolsErrorCategoryInvalidFlags)
			}

			// JSON output must be fully non-interactive: every choice has to come
			// from flags so no scope prompt or agent picker is shown.
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
					return withCategory(fmt.Errorf("--output json requires %s so the command runs without interactive prompts", strings.Join(missing, " and ")), protos.AitoolsErrorCategoryInvalidFlags)
				}
			}
			// A scope flag sets the scope here, so --project with --global is a
			// flag error; the scope prompt below runs only when no flag is set.
			var scope string
			if projectFlag || globalFlag {
				scope, err = resolveScope(projectFlag, globalFlag)
				if err != nil {
					return withCategory(err, protos.AitoolsErrorCategoryInvalidFlags)
				}
			}

			// Recovery steps repeat this run with the choices made so far, so a
			// copied command never widens to every agent or the global scope.
			// Agents chosen by flag or in the picker are named; without a
			// choice the re-run detects the same agents again.
			rerun := installInvocation{
				scope:        scope,
				skills:       skillsFlag,
				skillsOnly:   skillsOnly,
				experimental: includeExperimental,
				jsonOutput:   jsonMode,
			}

			// Validate --agents with the other flags, before the scope prompt.
			var targetAgents []*agents.Agent
			explicit := agentsFlag != ""
			if explicit {
				targetAgents, err = resolveAgentNames(ctx, agentsFlag)
				if err != nil {
					return err
				}
				funnel.setTargets(targetAgents)
				rerun.agents = joinAgentNames(targetAgents)
			}
			funnel.step(ctx, protos.AitoolsCommandStepFlagsValidated)

			// A person may be at this terminal with an agent env var inherited
			// from an agent session, so say why the prompts they would expect
			// were skipped.
			if !interactive && cmdio.IsPromptSupported(ctx) {
				var missing []string
				if scope == "" {
					missing = append(missing, "--scope")
				}
				if !explicit {
					missing = append(missing, "--agents")
				}
				if len(missing) > 0 {
					cmdio.LogString(ctx, agentCallerNote(missing))
				}
			}

			if scope == "" {
				scope, err = resolveScopeWithPrompt(ctx, interactive)
				if err != nil {
					return withRecovery(ctx, rerun, err)
				}
				rerun.scope = scope
			}
			opts.Scope = scope
			funnel.setScope(scope)
			funnel.step(ctx, protos.AitoolsCommandStepScopeResolved)

			if !explicit {
				targetAgents, err = selectAgents(ctx, scope, skillsOnly, interactive)
				if err != nil {
					return withRecovery(ctx, rerun, err)
				}
				if len(targetAgents) == 0 {
					printNoAgentsMessage(ctx, rerun)
					// Nothing to act on, like an empty --path dump, but the run
					// still fails, so a caller does not take it for a setup.
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeNothingToDo, protos.AitoolsErrorCategoryUnspecified)
					return errNoAgentSetUp
				}
				if interactive {
					rerun.agents = joinAgentNames(targetAgents)
				}
			}

			plan := buildPlan(ctx, targetAgents, scope, skillsOnly, explicit)
			funnel.setAgents(plan)
			funnel.step(ctx, protos.AitoolsCommandStepAgentsSelected)
			// The picker selection confirms the plan; runs without the picker
			// confirm it implicitly. The picker labels show which agents get
			// skill files instead of the plugin.
			funnel.step(ctx, protos.AitoolsCommandStepPlanConfirmed)
			if !explicit && interactive {
				printInstallScope(ctx, scope)
			}
			// Only a run past this point waits for the telemetry auth. A run that
			// ended earlier (a flag error, a cancelled prompt, no agent detected,
			// a --path dump) uploads its events only if auth had already resolved,
			// so it stays instant. The funnel therefore under-counts the fastest
			// early exits (which usually end before auth resolves), and their
			// STARTED events, relative to slower ones such as a cancelled prompt.
			auth.waitAtExit = true

			var outcomes []agentOutcome
			var runErr error
			defer func() {
				logInstallEvent(ctx, plan, installOpts{
					Scope:        opts.Scope,
					Experimental: opts.IncludeExperimental,
				}, classifyInstallError(topLevelFailure(runErr)), outcomes)
			}()

			outcomes, runErr = executePlan(ctx, src, plan, opts, rerun, jsonMode)
			for _, s := range deliverySteps(outcomes) {
				funnel.step(ctx, s)
			}
			switch {
			case runErr == nil:
				// The deferred funnel.finish records SUCCESS; a kept plugin counts
				// like an installed one, unless nothing else was done.
				if onlyKeptPlugins(outcomes) {
					funnel.finishWith(ctx, protos.AitoolsCommandOutcomeNothingToDo, protos.AitoolsErrorCategoryUnspecified)
				}
			case topLevelFailure(runErr) == nil:
				// Named agents failed, or no agent was set up. A skip has no error
				// type of its own, so take the category from the outcomes, not runErr.
				funnel.finishWith(ctx, protos.AitoolsCommandOutcomeFailure, firstFailureCategory(outcomes))
			default:
				// Finish on the real failure here: JSON mode returns the
				// ErrAlreadyPrinted sentinel below.
				funnel.finish(ctx, runErr)
			}

			if jsonMode {
				if jerr := renderJSON(cmd.OutOrStdout(), buildInstallOutput(ctx, rerun, opts.Scope, outcomes, runErr)); jerr != nil {
					// Rendering failed, so the JSON the caller parses is broken.
					// Report the render error unless the run already failed for
					// another reason.
					if runErr == nil {
						runErr = jerr
					}
					return runErr
				}
				// The JSON payload is the only thing on stdout and already reports
				// the outcome. On failure, exit non-zero without root printing a
				// duplicate "Error: ..." line to stderr; root prints errors itself
				// (see cmd/root/root.go), so ErrAlreadyPrinted is how a command
				// opts out of that
				if runErr != nil {
					return root.ErrAlreadyPrinted
				}
				return nil
			}
			return withRecovery(ctx, rerun, runErr)
		},
	}

	cmd.Flags().StringVar(&skillsFlag, "skills", "", "Specific skills to install (comma-separated)")
	cmd.Flags().StringVar(&agentsFlag, "agents", "", "Agents to install for (comma-separated, e.g. claude-code,cursor)")
	cmd.Flags().BoolVar(&includeExperimental, "experimental", false, "Include experimental skills")
	cmd.Flags().BoolVar(&skillsOnly, "skills-only", false, "Force raw skill files for every agent instead of the plugin (not with --path)")
	cmd.Flags().StringVar(&pathFlag, "path", "", "Write resolved skill files to this directory instead of using --skills-only (no agents, no state)")
	cmd.Flags().StringVar(&scopeFlag, "scope", "", "Install scope: project or global (default: global, or prompt when interactive)")
	cmd.Flags().BoolVar(&projectFlag, "project", false, "Install to project directory (cwd)")
	cmd.Flags().BoolVar(&globalFlag, "global", false, "Install globally (default)")
	markScopeBoolsDeprecated(cmd)
	// Resolve auth best-effort and time-bounded so telemetry can upload.
	auth = withTelemetryAuth(cmd)
	return cmd
}

// installOutputIsJSON reports whether --output json was requested. Unlike list,
// install can run detached from root: the legacy `skills install` alias builds a
// NewInstallCmd and executes it directly (see newLegacySkillsInstallCmd), so the
// root-supplied --output flag may be absent. Treat a missing flag as text rather
// than panicking the way root.OutputType would.
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
func selectAgents(ctx context.Context, scope string, skillsOnly, interactive bool) ([]*agents.Agent, error) {
	// Interactive: the picker decides; a prompt error or empty selection is a real
	// error, not a "nothing detected" no-op.
	if interactive {
		choices := agentChoices(ctx, scope, skillsOnly)
		if len(choices) == 0 {
			// Agents were detected but none can be acted on in this scope; the
			// caller prints the no-agents message rather than showing an empty picker.
			return nil, nil
		}
		return promptAgentSelection(ctx, scope, choices)
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
		item := planItemFor(ctx, a, scope, skillsOnly, false)
		label := agentChoiceLabel(ctx, a, item, skillsOnly)
		cmdio.LogString(ctx, fmt.Sprintf("  %-16s %s", a.DisplayName, label))
		if item.delivery == deliverySkip {
			continue
		}
		choices = append(choices, agentChoice{
			agent: a,
			label: a.DisplayName + "  " + label,
			// Under --skills-only, an agent whose config dir exists also starts
			// checked, so a plugin agent whose CLI is not on PATH is included
			// as in the non-interactive run. The usual pre-checks still apply.
			preselect: a.IsPreselected(ctx) || (skillsOnly && a.Detected(ctx)),
		})
	}
	return choices
}

// agentChoiceLabel is the picker label: the detection state, plus the skip
// reason when the agent can't be acted on in the chosen scope. Under
// --skills-only every found agent gets raw skills, so it is labeled "skills".
// An agent whose CLI is missing shows the skill files it gets instead of the
// plugin, or that it keeps the plugin it already has.
func agentChoiceLabel(ctx context.Context, a *agents.Agent, item agentPlanItem, skillsOnly bool) string {
	state := a.DisplayState(ctx)
	label := agentStateLabel(state)
	if skillsOnly && state != agents.StateNotFound {
		label = deliverySkills.String()
	}
	switch {
	case item.delivery == deliverySkip:
		return label + " · " + item.reason
	case item.cliMissing && item.delivery == deliverySkills:
		return "skills · CLI not found"
	case item.cliMissing && item.delivery == deliveryPlugin:
		return "plugin installed · CLI not found"
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

func defaultPromptAgentSelection(_ context.Context, scope string, choices []agentChoice) ([]*agents.Agent, error) {
	options := make([]huh.Option[string], 0, len(choices))
	byName := make(map[string]*agents.Agent, len(choices))
	for _, c := range choices {
		options = append(options, huh.NewOption(c.label, c.agent.Name).Selected(c.preselect))
		byName[c.agent.Name] = c.agent
	}

	var selected []string
	err := huh.NewMultiSelect[string]().
		Title(agentPickerTitle(scope)).
		Description("space to toggle, enter to install").
		Options(options...).
		Value(&selected).
		Run()
	if err != nil {
		return nil, promptError(err)
	}
	if len(selected) == 0 {
		return nil, errNoAgentsSelected
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
func buildPlan(ctx context.Context, targetAgents []*agents.Agent, scope string, skillsOnly, explicit bool) []agentPlanItem {
	plan := make([]agentPlanItem, 0, len(targetAgents))
	for _, a := range targetAgents {
		plan = append(plan, planItemFor(ctx, a, scope, skillsOnly, explicit))
	}
	return plan
}

// planItemFor resolves the delivery and scope for a single agent in the given
// install scope. It is shared by buildPlan and the interactive picker so the
// picker and the plan agree on what an agent will (or won't) do.
func planItemFor(ctx context.Context, a *agents.Agent, scope string, skillsOnly, explicit bool) agentPlanItem {
	item := agentPlanItem{agent: a, explicit: explicit}
	switch {
	case skillsOnly || a.Plugin == nil:
		// Raw-skills delivery (no-plugin agents, or --skills-only). Only some agents
		// support project-scoped skills, so skip the rest up front instead of
		// offering an option that fails at install time.
		if scope == installer.ScopeProject && !a.SupportsProjectScope {
			item.delivery = deliverySkip
			item.reason = "does not support project-scoped skills"
			item.skipErrorCategory = protos.AitoolsErrorCategoryUnsupportedScope
		} else {
			item.delivery = deliverySkills
		}
	default:
		nativeScope, ok, reason := mapAgentScope(a, scope)
		switch {
		case !ok:
			item.delivery = deliverySkip
			item.reason = reason
			item.skipErrorCategory = protos.AitoolsErrorCategoryUnsupportedScope
		case !explicit && !a.HasBinary(ctx) && a.Detected(ctx):
			// A desktop or IDE-only install has the config dir but no CLI to
			// install the plugin with, so it gets the skill files it can use
			// today. If it already has the plugin, skill files next to it would
			// show every skill twice, so the plugin is kept. A named agent keeps
			// failing with the commands to run: --agents never changes delivery.
			item.cliMissing = true
			item.delivery = deliverySkills
			switch {
			case pluginAlreadyInstalledFn(ctx, a, scope):
				item.delivery = deliveryPlugin
				item.scope = nativeScope
			default:
				// The skill files still serve every other project (global
				// scope) or the project's teammates (project scope), but a
				// plugin that does not cover this scope and applies here shows
				// every skill twice in this directory.
				item.projectPlugin = projectPluginActive(ctx, a)
			}
		default:
			item.delivery = deliveryPlugin
			item.scope = nativeScope
		}
	}
	return item
}

// pluginAlreadyInstalled reports whether the agent already has the databricks
// plugin for this install scope. When the CLI can read the agent's own plugin
// manifest, the manifest decides, as in agents.DatabricksTooling: a plugin
// removed with the agent's own CLI leaves the CLI's record behind. Otherwise,
// or when the manifest exists but cannot be read, the CLI's install state for
// the scope does.
func pluginAlreadyInstalled(ctx context.Context, a *agents.Agent, scope string) bool {
	if a.HasPluginManifestReader() {
		active, err := manifestPluginActive(ctx, a, scope)
		if err == nil {
			return active
		}
		log.Debugf(ctx, "Could not read the plugin manifest of %s, using the install state: %v", a.DisplayName, err)
	}
	return installer.PluginRecorded(ctx, scope, a.Name)
}

// manifestPluginActive reports whether the agent's own plugin manifest has a
// databricks plugin that serves this install scope. A global install serves
// every project, so only a machine-wide plugin covers it. A project install is
// shared through the repository, so only a project-scope plugin of this project
// covers it: a machine-wide or local plugin is personal, and teammates would
// get neither the plugin nor the skill files.
func manifestPluginActive(ctx context.Context, a *agents.Agent, scope string) (bool, error) {
	if scope != installer.ScopeProject {
		return a.DatabricksPluginActiveIn(ctx, "")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return false, err
	}
	return a.DatabricksProjectPluginIn(ctx, cwd)
}

// projectPluginActive reports whether the agent's own manifest has a databricks
// plugin record that applies in the current directory. Called after a global
// install found no machine-wide plugin, so a match is a project or local one:
// unlike manifestPluginActive for a project install, a local plugin counts,
// since its user sees both. An unreadable manifest or working directory
// reports false.
func projectPluginActive(ctx context.Context, a *agents.Agent) bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	active, err := a.DatabricksPluginActiveIn(ctx, cwd)
	return err == nil && active
}

// agentPickerTitle names the install scope, so it is visible before anything
// is installed.
func agentPickerTitle(scope string) string {
	return "Select agents to set up (" + scope + " scope)"
}

// printInstallScope prints the install scope after the agent picker, above the
// per-agent install lines.
func printInstallScope(ctx context.Context, scope string) {
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Install scope: "+scope)
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
	recovery      string                      // next step for the user, when skipped or failed
}

type outcomeStatus string

const (
	outcomeInstalled outcomeStatus = "installed"
	outcomeSkipped   outcomeStatus = "skipped"
	outcomeFailed    outcomeStatus = "failed"
	// outcomeKept is an agent whose CLI is missing but which already has the
	// plugin: it is set up, though nothing was installed.
	outcomeKept outcomeStatus = "kept"
)

// errNoAgentSetUp is returned when no agent was set up: none was detected, or
// every agent in the plan was skipped or failed. The reason and recovery step
// are printed above it, so the message only has to make the exit code non-zero.
var errNoAgentSetUp = errors.New("nothing was installed; follow the steps above")

// agentErrors wraps failures that are already reported in the per-agent
// outcomes: those of agents named via --agents, and errNoAgentSetUp when no
// detected agent was set up. Wrapping lets the JSON and telemetry
// layers tell a per-agent failure apart from a top-level failure that has no
// per-agent entry, so each is surfaced exactly once.
type agentErrors struct{ err error }

func (e *agentErrors) Error() string { return e.err.Error() }
func (e *agentErrors) Unwrap() error { return e.err }

// cliMissingOutcome is the plugin outcome of an agent whose CLI is not on PATH:
// failed when it got skill files instead, kept when it already has the plugin.
func cliMissingOutcome(it agentPlanItem, status outcomeStatus) agentOutcome {
	return agentOutcome{
		agent:         it.agent,
		delivery:      deliveryPlugin,
		status:        status,
		errorCategory: protos.AitoolsErrorCategoryCLINotOnPath,
		message:       cliMissingReason(it),
	}
}

// skillsBatchRerun reports the invocation that retries the skills batch when
// it included an agent whose CLI is missing and the run named its agents (the
// picker selection). Re-running with that agent named would try its plugin,
// since --agents never changes delivery, so it re-runs the skills agents with
// --skills-only instead. ok is false when rerun already retries the batch.
func skillsBatchRerun(rerun installInvocation, skillsAgents []*agents.Agent, fallbackItems []agentPlanItem) (installInvocation, bool) {
	if rerun.agents == "" || len(fallbackItems) == 0 {
		return rerun, false
	}
	rerun.agents = joinAgentNames(skillsAgents)
	rerun.skillsOnly = true
	return rerun, true
}

// skillsBatchError attaches the skillsBatchRerun recovery step to a failed
// skills batch. When there is no such step, the error is returned as is, and
// the caller attaches the step for the whole run.
func skillsBatchError(ctx context.Context, rerun installInvocation, skillsAgents []*agents.Agent, fallbackItems []agentPlanItem, err error) error {
	batchRerun, ok := skillsBatchRerun(rerun, skillsAgents, fallbackItems)
	if !ok {
		return err
	}
	return withRecovery(ctx, batchRerun, err)
}

// forgetStalePluginRecords drops the CLI's plugin records of agents that just
// got skill files instead of the plugin. They have no plugin (see
// pluginAlreadyInstalled), so a record left by a removal with the agent's own
// CLI is stale, and list and update would keep treating them as plugin agents
// and never refresh their skill files. The skills are already installed, so a
// failure here is only a warning.
func forgetStalePluginRecords(ctx context.Context, scope string, items []agentPlanItem) {
	if len(items) == 0 {
		return
	}
	names := make([]string, len(items))
	for i, it := range items {
		names[i] = it.agent.Name
	}
	if err := installer.ForgetPluginInstalls(ctx, scope, names); err != nil {
		log.Warnf(ctx, "Could not drop stale plugin records for %s: %v", strings.Join(names, ", "), err)
	}
}

// topLevelFailure returns the run error when it has no per-agent entry, or nil
// when the failure is already reported per agent — so a per-agent failure is not
// duplicated in the top-level error/errorCategory fields.
func topLevelFailure(runErr error) error {
	if _, ok := errors.AsType[*agentErrors](runErr); ok {
		return nil
	}
	return runErr
}

// executePlan carries out the plan and returns each agent's outcome. Skills
// installs, including those of agents whose CLI is missing, go through the
// existing skills path as one group. A failed plugin install is a warning,
// unless the agent was explicitly named via --agents, which is an error. A run
// that sets up no agent at all is an error too, so callers never mistake it for
// success. The per-agent recovery steps repeat rerun for the agent at hand.
func executePlan(ctx context.Context, src installer.ManifestSource, plan []agentPlanItem, opts installer.InstallOptions, rerun installInvocation, quiet bool) ([]agentOutcome, error) {
	var skillsAgents []*agents.Agent
	var fallbackItems, keptItems, pluginItems, skipItems []agentPlanItem
	skillsExplicit := false
	for _, it := range plan {
		switch it.delivery {
		case deliverySkills:
			skillsAgents = append(skillsAgents, it.agent)
			skillsExplicit = skillsExplicit || it.explicit
			if it.cliMissing {
				fallbackItems = append(fallbackItems, it)
			}
		case deliveryPlugin:
			if it.cliMissing {
				keptItems = append(keptItems, it)
			} else {
				pluginItems = append(pluginItems, it)
			}
		case deliverySkip:
			skipItems = append(skipItems, it)
		}
	}

	var outcomes []agentOutcome
	var explicitErrs []error

	// The plugin could not be installed for these agents. Keep that in the
	// outcomes so telemetry still sees it; the skills outcome records the
	// fallback.
	for _, it := range fallbackItems {
		if !quiet {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, skillsFallbackText(rerun, it)))
			if it.projectPlugin {
				cmdio.LogString(ctx, cmdio.Yellow(ctx, projectPluginWarning(opts.Scope, it)))
			}
		}
		outcomes = append(outcomes, cliMissingOutcome(it, outcomeFailed))
	}

	for _, it := range keptItems {
		if !quiet {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, pluginKeptText(rerun, it)))
		}
		// Raw skills an earlier run gave this agent would show every skill
		// twice next to the plugin, so remove them as a plugin install does.
		// This runs before the skills batch: a newer release there replaces the
		// recorded checksums, and older copies would then look user-modified
		// and stay. Only the agent's own manifest is proof enough: for Codex and
		// Copilot the CLI's record is the only evidence, and it outlives a
		// removal with the agent's own CLI, so removing the skills then would
		// leave the agent with neither.
		if active, err := manifestPluginActive(ctx, it.agent, opts.Scope); err == nil && active {
			if err := cleanupLegacyFn(ctx, it.agent, opts.Scope); err != nil {
				log.Debugf(ctx, "Legacy skill cleanup for %s failed: %v", it.agent.DisplayName, err)
			}
		}
		outcomes = append(outcomes, cliMissingOutcome(it, outcomeKept))
	}

	// A skills install runs as a group; its failure is the command's top-level
	// error. Plugin agents don't depend on it, so they are still attempted.
	var skillsErr error
	if len(skillsAgents) > 0 {
		if !quiet {
			installer.PrintInstallingFor(ctx, skillsAgents)
		}
		skillsErr = installSkillsForAgentsFn(ctx, src, skillsAgents, opts)
		if skillsErr == nil {
			for _, a := range skillsAgents {
				outcomes = append(outcomes, agentOutcome{agent: a, delivery: deliverySkills, status: outcomeInstalled})
			}
			forgetStalePluginRecords(ctx, opts.Scope, fallbackItems)
		}
	}

	pluginCount := 0
	pluginsRecorded := false
	if len(pluginItems) > 0 {
		// Plugins install from the agent's marketplace, so the skills ref only
		// labels the recorded version. Record it as unknown when unresolved.
		ref, _, err := installer.GetSkillsRef(ctx)
		if err != nil {
			log.Debugf(ctx, "Recording plugin version as unknown: %v", err)
			ref = ""
		}
		records := map[string]installer.PluginRecord{}
		for _, it := range pluginItems {
			if !quiet {
				cmdio.LogString(ctx, fmt.Sprintf("Installing databricks plugin for %s...", it.agent.DisplayName))
			}
			rec, err := installPluginForAgentFn(ctx, it.agent, it.scope, ref)
			if err != nil {
				reason, recovery := pluginFailureText(ctx, rerun, it, err)
				if !quiet {
					cmdio.LogString(ctx, cmdio.Yellow(ctx, fmt.Sprintf("Skipped %s: %s", it.agent.DisplayName, reason)))
					cmdio.LogString(ctx, "  "+recovery)
				}
				outcomes = append(outcomes, agentOutcome{
					agent:         it.agent,
					delivery:      deliveryPlugin,
					status:        outcomeFailed,
					errorCategory: classifyInstallError(err),
					message:       err.Error(),
					recovery:      recovery,
				})
				if it.explicit {
					explicitErrs = append(explicitErrs, &displayError{msg: it.agent.DisplayName + ": " + reason, err: err})
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
		if len(records) > 0 && skillsErr != nil && installer.HasUntrackedSkills(ctx, opts.Scope) {
			// Skills with no state file are on disk (an install that predates
			// state tracking, or an interrupted first install). A state file
			// holding only these plugins would hide those skills from update and
			// from the install's legacy checks, so leave recording to the next
			// install whose skills step succeeds.
			log.Debugf(ctx, "Not recording plugin installs: skills without install state are on disk")
		} else if len(records) > 0 {
			// After a skills failure the skills on disk stay at the old release,
			// so record the plugins without advancing state.Release.
			releaseRef := ref
			if skillsErr != nil {
				releaseRef = ""
			}
			if err := recordPluginInstallsFn(ctx, opts.Scope, records, releaseRef); err != nil {
				// No skill to install is no failure of its own; report only the
				// record failure.
				if errors.Is(skillsErr, installer.ErrNoSkillsResolved) {
					return outcomes, err
				}
				return outcomes, errors.Join(skillsBatchError(ctx, rerun, skillsAgents, fallbackItems, skillsErr), err)
			}
			pluginsRecorded = true
		}
	}

	for _, it := range skipItems {
		recovery := skipRecovery(rerun, it)
		if !quiet {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, "Skipped "+it.agent.DisplayName+": "+it.reason))
			cmdio.LogString(ctx, "  "+recovery)
		}
		outcomes = append(outcomes, agentOutcome{
			agent:         it.agent,
			delivery:      deliverySkip,
			status:        outcomeSkipped,
			errorCategory: it.skipErrorCategory,
			message:       it.reason,
			recovery:      recovery,
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

	// A release with no skill this CLI can install skips the skills agents when
	// the plan had plugin agents, installed or kept, so the plugin outcomes
	// decide the run. A --skills-only run with nothing else attempted keeps the
	// error. So does a run whose installed plugins were not recorded (skills
	// without install state are on disk): update and uninstall could not
	// manage them.
	if errors.Is(skillsErr, installer.ErrNoSkillsResolved) && (len(pluginItems) > 0 || len(keptItems) > 0) && (pluginCount == 0 || pluginsRecorded) {
		batchRerun, _ := skillsBatchRerun(rerun, skillsAgents, fallbackItems)
		recovery := installRecovery(ctx, batchRerun, skillsErr)
		names := make([]string, len(skillsAgents))
		for i, a := range skillsAgents {
			names[i] = a.DisplayName
			outcomes = append(outcomes, agentOutcome{agent: a, delivery: deliverySkills, status: outcomeSkipped, errorCategory: classifyInstallError(skillsErr), message: skillsErr.Error(), recovery: recovery})
		}
		if !quiet {
			cmdio.LogString(ctx, cmdio.Yellow(ctx, "Skipped "+strings.Join(names, ", ")+": "+skillsErr.Error()))
			cmdio.LogString(ctx, "  "+recovery)
		}
		if skillsExplicit {
			explicitErrs = append(explicitErrs, &displayError{msg: strings.Join(names, ", ") + ": " + skillsErr.Error(), err: skillsErr})
		}
		skillsErr = nil
	}

	// The skills error has no per-agent entry, so it stays the top-level error;
	// explicit per-agent failures are already in their outcomes.
	if skillsErr != nil {
		return outcomes, skillsBatchError(ctx, rerun, skillsAgents, fallbackItems, skillsErr)
	}
	if len(explicitErrs) > 0 {
		return outcomes, &agentErrors{err: errors.Join(explicitErrs...)}
	}
	if !anyInstalled(outcomes) {
		return outcomes, &agentErrors{err: errNoAgentSetUp}
	}
	return outcomes, nil
}

type installOutput struct {
	Scope  string            `json:"scope"`
	Agents []agentResultJSON `json:"agents"`

	// Error and ErrorCategory describe a top-level failure with no per-agent
	// entry (e.g. a skills-group install failure); both empty on success. Error
	// is the local-only message; ErrorCategory is the classification telemetry
	// also records.
	Error         string `json:"error,omitempty"`
	ErrorCategory string `json:"error_category,omitempty"`
	// Recovery is the next step for the top-level failure, local-only like Error.
	Recovery string `json:"recovery,omitempty"`
}

type agentResultJSON struct {
	Name          string `json:"name"`
	Delivery      string `json:"delivery"`
	Status        string `json:"status"`
	ErrorCategory string `json:"error_category,omitempty"`
	Message       string `json:"message,omitempty"`
	Recovery      string `json:"recovery,omitempty"`
}

// buildInstallOutput builds the --output json result; the recovery steps
// repeat rerun.
func buildInstallOutput(ctx context.Context, rerun installInvocation, scope string, outcomes []agentOutcome, runErr error) installOutput {
	out := installOutput{Scope: scope, Agents: make([]agentResultJSON, 0, len(outcomes))}
	for _, o := range outcomes {
		entry := agentResultJSON{
			Name:     o.agent.Name,
			Delivery: o.delivery.String(),
			Status:   string(o.status),
			Message:  o.message,
			Recovery: o.recovery,
		}
		// Only non-successful outcomes carry a category, keyed on status to match
		// agentResultsField in telemetry.go so the JSON and telemetry views of the
		// same slice never disagree.
		if o.status != outcomeInstalled {
			entry.ErrorCategory = string(o.errorCategory)
		}
		out.Agents = append(out.Agents, entry)
	}
	// A top-level failure (skills-group install, plugin recording) has no
	// per-agent entry, so surface it here too; otherwise the consumer sees a
	// non-zero exit with no reason, and the agents array holds at most the
	// plugin agents that were still installed. Per-agent failures stay in the
	// agents entries above and are not repeated here.
	if e := topLevelFailure(runErr); e != nil {
		out.Error = e.Error()
		out.ErrorCategory = string(classifyInstallError(e))
		out.Recovery = installRecovery(ctx, rerun, e)
	}
	return out
}

// joinAgentNames is the --agents value that names targetAgents.
func joinAgentNames(targetAgents []*agents.Agent) string {
	names := make([]string, len(targetAgents))
	for i, a := range targetAgents {
		names[i] = a.Name
	}
	return strings.Join(names, ",")
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
			return nil, withCategory(fmt.Errorf("unknown agent %q. Available agents: %s", name, strings.Join(availableNames, ", ")), protos.AitoolsErrorCategoryUnknownAgent)
		}
		result = append(result, agent)
	}

	if len(result) == 0 {
		return nil, withCategory(errors.New("no agents specified"), protos.AitoolsErrorCategoryInvalidFlags)
	}
	return result, nil
}

// printNoAgentsMessage prints the "no agents detected" message. Its commands
// repeat rerun, so a named agent is set up in the scope and mode this run used.
func printNoAgentsMessage(ctx context.Context, rerun installInvocation) {
	// --skills-only detects an agent by its config dir only, not by its CLI.
	found := "No supported coding agents found on PATH."
	if rerun.skillsOnly {
		found = "No supported coding agent config directories found."
	}
	cmdio.LogString(ctx, cmdio.Yellow(ctx, found))
	cmdio.LogString(ctx, "")
	cmdio.LogString(ctx, "Supported: "+strings.Join(agents.SupportedNames(), ", ")+".")
	named, _ := commandText(runtime.GOOS, rerun, nil, "--agents=<name>")
	cmdio.LogString(ctx, "Install one, then re-run "+rerunPhrase(runtime.GOOS, rerun)+". To set up an agent that was not detected, name it: "+named)
}
