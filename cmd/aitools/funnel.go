package aitools

import (
	"context"
	"errors"
	"slices"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/aitools/installer"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
	"github.com/spf13/cobra"
)

// commandFunnel emits the AitoolsCommandEvent funnel of one aitools subcommand
// run: STARTED when created, STEP_COMPLETED per step, and exactly one FINISHED.
// The scope from setScope is attached to every later event; the agents from
// setAgents only to FINISHED.
type commandFunnel struct {
	command     protos.AitoolsCommand
	interactive bool
	installMode protos.AitoolsInstallMode
	scope       protos.AitoolsInstallScope
	agents      []protos.AitoolsAgentType
	finished    bool
}

// startFunnel logs the STARTED event for command and returns the funnel to
// report later steps and the outcome on.
// installMode is set for install only, so it is on the STARTED event too.
// interactive is whether the command may show prompts.
func startFunnel(ctx context.Context, command protos.AitoolsCommand, installMode protos.AitoolsInstallMode, interactive bool) *commandFunnel {
	f := &commandFunnel{
		command:     command,
		interactive: interactive,
		installMode: installMode,
	}
	f.log(ctx, protos.AitoolsCommandEvent{Stage: protos.AitoolsCommandStageStarted})
	return f
}

// setScope records the resolved scope (installer.ScopeGlobal/ScopeProject).
// An empty scope (none or both selected) leaves the field unset, so the event
// omits it rather than sending the Unspecified sentinel.
func (f *commandFunnel) setScope(scope string) {
	if scope == "" {
		f.scope = ""
		return
	}
	f.scope = scopeType(scope)
}

// setAgents records the plan's agents, reported on the FINISHED event.
func (f *commandFunnel) setAgents(plan []agentPlanItem) { f.agents = agentsField(plan) }

// setTargets records agents known before the plan is built (named via
// --agents), so a run that stops at the scope prompt still reports them.
func (f *commandFunnel) setTargets(targets []*agents.Agent) { f.agents = agentTypes(targets) }

// step logs a STEP_COMPLETED event.
func (f *commandFunnel) step(ctx context.Context, s protos.AitoolsCommandStep) {
	f.log(ctx, protos.AitoolsCommandEvent{
		Stage: protos.AitoolsCommandStageStepCompleted,
		Step:  s,
	})
}

// finish logs the FINISHED event for a run that returned err: SUCCESS for nil,
// NOTHING_TO_DO when update or uninstall found nothing installed, or an install
// found no skill in the release it can install (the command still exits
// non-zero), CANCELLED when the user aborted a prompt or picked no agents,
// FAILURE otherwise. No-op once the run has finished, so it is safe to defer
// after an explicit finishWith.
func (f *commandFunnel) finish(ctx context.Context, err error) {
	switch {
	case err == nil:
		f.finishWith(ctx, protos.AitoolsCommandOutcomeSuccess, protos.AitoolsErrorCategoryUnspecified)
	case errors.Is(err, installer.ErrNothingInstalled), errors.Is(err, installer.ErrNoSkillsResolved):
		f.finishWith(ctx, protos.AitoolsCommandOutcomeNothingToDo, protos.AitoolsErrorCategoryUnspecified)
	case errors.Is(err, huh.ErrUserAborted):
		f.finishWith(ctx, protos.AitoolsCommandOutcomeCancelled, protos.AitoolsErrorCategoryUserAborted)
	case errors.Is(err, errNoAgentsSelected):
		f.finishWith(ctx, protos.AitoolsCommandOutcomeCancelled, protos.AitoolsErrorCategoryUnspecified)
	default:
		f.finishWith(ctx, protos.AitoolsCommandOutcomeFailure, classifyInstallError(err))
	}
}

// finishWith logs the FINISHED event with an explicit outcome, for runs whose
// returned error does not tell it: a nil return without success (a declined
// confirm, nothing to act on), a failure already reported per agent, or a run
// with no agent detected, which fails but had nothing to act on.
func (f *commandFunnel) finishWith(ctx context.Context, outcome protos.AitoolsCommandOutcome, category protos.AitoolsErrorCategory) {
	if f.finished {
		return
	}
	f.finished = true
	// Unspecified means "no error"; omit it rather than send the sentinel.
	if category == protos.AitoolsErrorCategoryUnspecified {
		category = ""
	}
	f.log(ctx, protos.AitoolsCommandEvent{
		Stage:         protos.AitoolsCommandStageFinished,
		Outcome:       outcome,
		ErrorCategory: category,
		Agents:        f.agents,
	})
}

// log stamps e with the run's command, interactivity, install mode and scope,
// and buffers it for upload.
func (f *commandFunnel) log(ctx context.Context, e protos.AitoolsCommandEvent) {
	e.Command = f.command
	e.Interactive = f.interactive
	e.InstallMode = f.installMode
	e.Scope = f.scope
	telemetry.Log(ctx, protos.DatabricksCliLog{AitoolsCommandEvent: &e})
}

// withCommandFunnel wraps a subcommand's RunE so every run logs STARTED and
// FINISHED (outcome from the returned error).
func withCommandFunnel(command protos.AitoolsCommand, runE func(cmd *cobra.Command, args []string) error) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) (err error) {
		f := startFunnel(cmd.Context(), command, "", cmdio.IsPromptSupported(cmd.Context()))
		defer func() { f.finish(cmd.Context(), err) }()
		cmd.SetContext(context.WithValue(cmd.Context(), funnelKey{}, f))
		return runE(cmd, args)
	}
}

type funnelKey struct{}

// funnelFrom returns the funnel withCommandFunnel put on ctx. Only the RunE of
// a command wrapped by instrumentCommand calls it, for outcomes its returned
// error does not tell (see finishWith) and for the scope and agents it
// resolves.
func funnelFrom(ctx context.Context) *commandFunnel {
	return ctx.Value(funnelKey{}).(*commandFunnel)
}

// installModeType maps the install delivery flags to the telemetry enum.
func installModeType(skillsOnly bool, path string) protos.AitoolsInstallMode {
	switch {
	case path != "":
		return protos.AitoolsInstallModePath
	case skillsOnly:
		return protos.AitoolsInstallModeSkillsOnly
	default:
		return protos.AitoolsInstallModeDefault
	}
}

// deliverySteps returns the delivery steps executePlan got through, derived
// from its outcomes: SKILLS_INSTALLED when the skills group installed, and
// PLUGINS_INSTALLED when at least one plugin installed. Events are
// uploaded together at exit, so logging them after executePlan returns yields
// the same payload as logging them inside it.
func deliverySteps(outcomes []agentOutcome) []protos.AitoolsCommandStep {
	var skills, plugins bool
	for _, o := range outcomes {
		switch o.delivery {
		case deliverySkills:
			skills = skills || o.status == outcomeInstalled
		case deliveryPlugin:
			plugins = plugins || o.status == outcomeInstalled
		case deliverySkip:
		}
	}
	var steps []protos.AitoolsCommandStep
	if skills {
		steps = append(steps, protos.AitoolsCommandStepSkillsInstalled)
	}
	if plugins {
		steps = append(steps, protos.AitoolsCommandStepPluginsInstalled)
	}
	return steps
}

// anyInstalled reports whether at least one agent was set up.
func anyInstalled(outcomes []agentOutcome) bool {
	return slices.ContainsFunc(outcomes, func(o agentOutcome) bool { return o.status == outcomeInstalled })
}

// firstFailureCategory returns the category of the first agent that failed or
// was skipped, or UNCATEGORIZED when none carries one.
func firstFailureCategory(outcomes []agentOutcome) protos.AitoolsErrorCategory {
	for _, o := range outcomes {
		if o.status != outcomeInstalled && o.errorCategory != "" && o.errorCategory != protos.AitoolsErrorCategoryUnspecified {
			return o.errorCategory
		}
	}
	return protos.AitoolsErrorCategoryUncategorized
}
