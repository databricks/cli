package aitools

import (
	"context"
	"errors"

	"github.com/charmbracelet/huh"
	"github.com/databricks/cli/libs/aitools/agents"
	"github.com/databricks/cli/libs/cmdio"
	"github.com/databricks/cli/libs/telemetry"
	"github.com/databricks/cli/libs/telemetry/protos"
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
func startFunnel(ctx context.Context, command protos.AitoolsCommand, installMode protos.AitoolsInstallMode) *commandFunnel {
	f := &commandFunnel{
		command:     command,
		interactive: cmdio.IsPromptSupported(ctx),
		installMode: installMode,
	}
	f.log(ctx, protos.AitoolsCommandEvent{Stage: protos.AitoolsCommandStageStarted})
	return f
}

// setScope records the resolved scope (installer.ScopeGlobal/ScopeProject).
func (f *commandFunnel) setScope(scope string) { f.scope = scopeType(scope) }

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
// CANCELLED when the user aborted a prompt or picked no agents, FAILURE
// otherwise. No-op once the run has finished, so it is safe to defer after an
// explicit finishWith.
func (f *commandFunnel) finish(ctx context.Context, err error) {
	switch {
	case err == nil:
		f.finishWith(ctx, protos.AitoolsCommandOutcomeSuccess, protos.AitoolsErrorCategoryUnspecified)
	case errors.Is(err, huh.ErrUserAborted):
		f.finishWith(ctx, protos.AitoolsCommandOutcomeCancelled, protos.AitoolsErrorCategoryUserAborted)
	case errors.Is(err, errNoAgentsSelected):
		f.finishWith(ctx, protos.AitoolsCommandOutcomeCancelled, protos.AitoolsErrorCategoryUnspecified)
	default:
		f.finishWith(ctx, protos.AitoolsCommandOutcomeFailure, classifyInstallError(err))
	}
}

// finishWith logs the FINISHED event with an explicit outcome, for runs whose
// returned error does not tell it: a nil return without success (declined
// confirm, nothing to act on, nothing installed) or a named agent's failure.
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

// nothingInstalledCategory reports whether a run that returned no error still
// installed nothing for any agent (e.g. the only detected agent's plugin
// install failed, which is a warning, not an error), and if so the category of
// the first agent that failed or was skipped. The funnel records such a run as
// FAILURE rather than SUCCESS. That includes a non-interactive run whose
// detected agents are all skipped for the scope (UNSUPPORTED_SCOPE): it set
// up nothing it was asked to. An interactive run never gets here for them,
// because the picker offers no skipped agent and the run finishes NOTHING_TO_DO
// before the plan.
func nothingInstalledCategory(outcomes []agentOutcome) (protos.AitoolsErrorCategory, bool) {
	if len(outcomes) == 0 {
		return "", false
	}
	for _, o := range outcomes {
		if o.status == outcomeInstalled {
			return "", false
		}
	}
	return firstFailureCategory(outcomes), true
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
