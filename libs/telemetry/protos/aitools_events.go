package protos

// AitoolsHintKind mirrors AitoolsHintKind.Type in the databricks_cli lumberjack
// proto: which Databricks AI tools hint was shown.
type AitoolsHintKind string

const (
	AitoolsHintKindUnspecified AitoolsHintKind = "TYPE_UNSPECIFIED"
	// AitoolsHintKindInstallText is the generic stderr line asking the agent to
	// run `databricks aitools install`.
	AitoolsHintKindInstallText AitoolsHintKind = "INSTALL_TEXT"
	// AitoolsHintKindClaudeCodeNative is Claude Code's native
	// `<claude-code-hint>` line, which makes Claude Code offer to install the
	// Databricks plugin.
	AitoolsHintKindClaudeCodeNative AitoolsHintKind = "CLAUDE_CODE_NATIVE"
	// AitoolsHintKindAgentInstallText is the stderr line asking a non-Claude
	// coding agent to run `databricks aitools install` for itself.
	AitoolsHintKindAgentInstallText AitoolsHintKind = "AGENT_INSTALL_TEXT"
	// AitoolsHintKindSkillsListText is the stderr line asking the agent to run
	// `databricks skills list`, shown in place of the install text (of either
	// kind) when skill hints are on.
	AitoolsHintKindSkillsListText AitoolsHintKind = "SKILLS_LIST_TEXT"
	// AitoolsHintKindAppsInitTip is the non-interactive tip printed by
	// `databricks apps init`.
	AitoolsHintKindAppsInitTip AitoolsHintKind = "APPS_INIT_TIP"
	// AitoolsHintKindAppsInitPrompt is the interactive "install skills now?"
	// prompt shown by `databricks apps init`.
	AitoolsHintKindAppsInitPrompt AitoolsHintKind = "APPS_INIT_PROMPT"
)

// AitoolsHintEvent is emitted once for each hint line the CLI prints
// for a coding agent (Claude Code's native and text lines are one event each)
// and for the `apps init` skills tip or install prompt; other messages that
// mention `databricks aitools install` log nothing. It carries no
// user-authored strings. The command it was shown on is ExecutionContext.Command
// of the enclosing log entry. The CLI sets no calling-agent field on this
// event: the calling agent is reported by the User-Agent of the telemetry
// upload (`agent/<name>`).
type AitoolsHintEvent struct {
	HintKind AitoolsHintKind `json:"hint_kind"`
}

// AitoolsCommand mirrors AitoolsCommand.Type: the aitools subcommand an
// AitoolsCommandEvent belongs to.
type AitoolsCommand string

const (
	AitoolsCommandUnspecified AitoolsCommand = "TYPE_UNSPECIFIED"
	AitoolsCommandInstall     AitoolsCommand = "INSTALL"
	AitoolsCommandUpdate      AitoolsCommand = "UPDATE"
	AitoolsCommandList        AitoolsCommand = "LIST"
	AitoolsCommandVersion     AitoolsCommand = "VERSION"
	AitoolsCommandUninstall   AitoolsCommand = "UNINSTALL"
)

// AitoolsCommandStage mirrors AitoolsCommandStage.Type: where in the command's
// funnel an AitoolsCommandEvent was emitted.
type AitoolsCommandStage string

const (
	AitoolsCommandStageUnspecified   AitoolsCommandStage = "TYPE_UNSPECIFIED"
	AitoolsCommandStageStarted       AitoolsCommandStage = "STARTED"
	AitoolsCommandStageStepCompleted AitoolsCommandStage = "STEP_COMPLETED"
	AitoolsCommandStageFinished      AitoolsCommandStage = "FINISHED"
)

// AitoolsCommandStep mirrors AitoolsCommandStep.Type: a funnel step of
// `aitools install`, in the order the command reaches them. A run that stops
// early has no events for the later steps, so the last STEP_COMPLETED event
// shows where it stopped.
type AitoolsCommandStep string

const (
	AitoolsCommandStepUnspecified AitoolsCommandStep = "TYPE_UNSPECIFIED"
	// Flag combinations were validated.
	AitoolsCommandStepFlagsValidated AitoolsCommandStep = "FLAGS_VALIDATED"
	// The install scope was resolved (from flags or the scope prompt).
	AitoolsCommandStepScopeResolved AitoolsCommandStep = "SCOPE_RESOLVED"
	// The target agents were resolved (from --agents, the picker, or detection).
	AitoolsCommandStepAgentsSelected AitoolsCommandStep = "AGENTS_SELECTED"
	// The plan was confirmed. Install has no separate confirm prompt: this step
	// follows AGENTS_SELECTED right away, because picking agents in the
	// interactive picker confirms the plan, and runs without the picker
	// (--agents, non-interactive) confirm it implicitly.
	AitoolsCommandStepPlanConfirmed AitoolsCommandStep = "PLAN_CONFIRMED"
	// Raw skill files were installed for the skills-delivery agents.
	AitoolsCommandStepSkillsInstalled AitoolsCommandStep = "SKILLS_INSTALLED"
	// At least one plugin-delivery agent got the plugin (others may still have
	// failed; the run's AitoolsInstallEvent agent_results has them per agent).
	AitoolsCommandStepPluginsInstalled AitoolsCommandStep = "PLUGINS_INSTALLED"
	// Skill files were written to --path.
	AitoolsCommandStepSkillsDumped AitoolsCommandStep = "SKILLS_DUMPED"
)

// AitoolsCommandOutcome mirrors AitoolsCommandOutcome.Type: how a command run
// ended. Only set on FINISHED events.
type AitoolsCommandOutcome string

const (
	AitoolsCommandOutcomeUnspecified AitoolsCommandOutcome = "TYPE_UNSPECIFIED"
	AitoolsCommandOutcomeSuccess     AitoolsCommandOutcome = "SUCCESS"
	AitoolsCommandOutcomeFailure     AitoolsCommandOutcome = "FAILURE"
	// The user declined the confirm prompt, aborted a prompt, or confirmed the
	// agent picker with nothing selected.
	AitoolsCommandOutcomeCancelled AitoolsCommandOutcome = "CANCELLED"
	// The command had nothing to act on (e.g. no coding agents detected, a
	// --path dump where every skill was filtered out, or nothing installed to
	// update or uninstall). The command can still exit non-zero.
	AitoolsCommandOutcomeNothingToDo AitoolsCommandOutcome = "NOTHING_TO_DO"
)

// AitoolsInstallMode mirrors AitoolsInstallMode.Type: how `aitools install`
// delivers the tools.
type AitoolsInstallMode string

const (
	AitoolsInstallModeUnspecified AitoolsInstallMode = "TYPE_UNSPECIFIED"
	// Plugin where the agent supports it, skills otherwise.
	AitoolsInstallModeDefault AitoolsInstallMode = "DEFAULT"
	// --skills-only: raw skill files for every agent.
	AitoolsInstallModeSkillsOnly AitoolsInstallMode = "SKILLS_ONLY"
	// --path: skill files written to a directory, no agents.
	AitoolsInstallModePath AitoolsInstallMode = "PATH"
)

// AitoolsCommandEvent is one funnel event of an aitools subcommand. Every run
// of install, update, list, version and uninstall that gets past flag parsing
// emits a STARTED event and a FINISHED event; install also emits a
// STEP_COMPLETED event per step reached. All events of one run share
// ExecutionContext.CmdExecID.
// The CLI sets no calling-agent field on this event: it reports the calling
// agent through the User-Agent of the telemetry upload (`agent/<name>`, and its
// Databricks tooling as `aitools-agent/...`), together with the same
// cmd-exec-id.
type AitoolsCommandEvent struct {
	Command AitoolsCommand      `json:"command"`
	Stage   AitoolsCommandStage `json:"stage"`

	// Set on STEP_COMPLETED events only.
	Step AitoolsCommandStep `json:"step,omitempty"`

	// Outcome and ErrorCategory are set on FINISHED events only. ErrorCategory
	// is set for FAILURE, and for CANCELLED only when a prompt was aborted
	// (USER_ABORTED); otherwise it is omitted.
	Outcome       AitoolsCommandOutcome `json:"outcome,omitempty"`
	ErrorCategory AitoolsErrorCategory  `json:"error_category,omitempty"`

	// Whether the command could prompt: cmdio.IsPromptSupported and no coding
	// agent detected, so an agent on a terminal reports false. The same for
	// every command, including list and version, which never prompt. A
	// measurable false, so no omitempty.
	Interactive bool `json:"interactive"`

	// Install only: delivery mode. Set on every install event once flags parse.
	InstallMode AitoolsInstallMode `json:"install_mode,omitempty"`

	// Install and uninstall: agents the run targeted (uninstall: from
	// --agents), deduplicated and sorted. Set on the FINISHED event once the
	// targets are known.
	Agents []AitoolsAgentType `json:"agents,omitempty"`

	// Install, update, list and uninstall: the single scope the run acts on.
	// Set on events once a scope flag selects one or the scope is resolved;
	// omitted when none or both scopes are selected.
	Scope AitoolsInstallScope `json:"scope,omitempty"`
}

// AitoolsAutoInstallOutcome mirrors AitoolsAutoInstallOutcome.Type: how the
// automatic installation of Databricks AI tools on the first run of a new CLI
// version ended.
type AitoolsAutoInstallOutcome string

const (
	AitoolsAutoInstallOutcomeUnspecified AitoolsAutoInstallOutcome = "TYPE_UNSPECIFIED"
	// The tools were installed, or an existing installation was updated and
	// changed.
	AitoolsAutoInstallOutcomeInstalled AitoolsAutoInstallOutcome = "INSTALLED"
	// An existing installation was updated and nothing changed: it was current,
	// or every outdated skill was kept because the user modified it.
	AitoolsAutoInstallOutcomeUpToDate AitoolsAutoInstallOutcome = "UP_TO_DATE"
	// The installed tools were left as they are: skills without install state
	// (installed by hand, or by an older CLI), or skills newer than the release
	// this CLI installs.
	AitoolsAutoInstallOutcomeKept AitoolsAutoInstallOutcome = "KEPT"
	// Every change would have run an agent CLI, which auto-install does not do
	// on Windows: a plan of only plugin installs, or an update of an
	// installation with plugins.
	AitoolsAutoInstallOutcomeSkippedPlatform AitoolsAutoInstallOutcome = "SKIPPED_PLATFORM"
	// The run was in CI (the CI env var, or a CI provider's own).
	AitoolsAutoInstallOutcomeSkippedCI AitoolsAutoInstallOutcome = "SKIPPED_CI"
	// The user turned auto-install off: DATABRICKS_AITOOLS_AUTO_INSTALL=false,
	// `aitools auto-install disable`, a global `aitools uninstall`, "no" at
	// the prompt of an earlier version, or "no" to the skills prompt of
	// `apps init`.
	AitoolsAutoInstallOutcomeSkippedOptOut AitoolsAutoInstallOutcome = "SKIPPED_OPTOUT"
	// The machine is in the holdout cohort.
	AitoolsAutoInstallOutcomeSkippedHoldout AitoolsAutoInstallOutcome = "SKIPPED_HOLDOUT"
	// No supported coding agent has a config directory on the machine.
	AitoolsAutoInstallOutcomeSkippedNoAgents AitoolsAutoInstallOutcome = "SKIPPED_NO_AGENTS"
	// The user answered "no" at the install prompt, which also turns
	// auto-install off.
	AitoolsAutoInstallOutcomeDeclined AitoolsAutoInstallOutcome = "DECLINED"
	// The install prompt got no answer: stdin was closed, Ctrl+C was pressed,
	// it timed out, or the answer was neither yes nor no. Nothing is installed
	// and auto-install stays on, so the next CLI version asks again.
	AitoolsAutoInstallOutcomeUnanswered AitoolsAutoInstallOutcome = "UNANSWERED"
	// The attempt failed or timed out; a later run of the same version
	// retries.
	AitoolsAutoInstallOutcomeFailed AitoolsAutoInstallOutcome = "FAILED"
	// The attempt failed and was the last one this CLI version gets.
	AitoolsAutoInstallOutcomeRetriesExhausted AitoolsAutoInstallOutcome = "RETRIES_EXHAUSTED"
)

// AitoolsHoldoutArm mirrors AitoolsHoldoutArm.Type: the auto-install
// experiment arm of the machine, from a stable per-machine bucket.
type AitoolsHoldoutArm string

const (
	AitoolsHoldoutArmUnspecified AitoolsHoldoutArm = "TYPE_UNSPECIFIED"
	// The machine never auto-installs (control group).
	AitoolsHoldoutArmHoldout AitoolsHoldoutArm = "HOLDOUT"
	// The machine auto-installs.
	AitoolsHoldoutArmTreatment AitoolsHoldoutArm = "TREATMENT"
)

// AitoolsAutoInstallEvent is emitted when the CLI decides about installing
// Databricks AI tools on the first run of a new version: once per version and
// machine, plus once per retry after a failed attempt. A version whose outcome
// is settled emits nothing, and so does a run that the remote gate turns off;
// the CI and opt-out checks come before the gate, so they are emitted either
// way. A run that has to ask before installing but cannot emits nothing and
// leaves the version to a later run. An install with no tools installed yet
// also emits an AitoolsInstallEvent with Trigger AUTO_FIRST_RUN.
type AitoolsAutoInstallEvent struct {
	Outcome    AitoolsAutoInstallOutcome `json:"outcome"`
	HoldoutArm AitoolsHoldoutArm         `json:"holdout_arm"`

	// Whether a person may be at the terminal (cmdio.HasTerminal and no
	// coding agent detected). Such a run asks before the first install of a
	// version, so DECLINED and UNANSWERED come only with true; INSTALLED with
	// true may still have run without asking (an update, or a retry after an
	// accepted attempt). A measurable false, so no omitempty.
	Interactive bool `json:"interactive"`

	// Set for FAILED and RETRIES_EXHAUSTED.
	ErrorCategory AitoolsErrorCategory `json:"error_category,omitempty"`
}
