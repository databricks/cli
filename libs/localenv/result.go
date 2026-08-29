package localenv

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Command path components, defined once so a rename touches a single place
// (spec §0 / invariant 8 / scenario 21). The verb is a subcommand of the
// generated "environments" group; the --json "command" field uses CommandName.
// No other string re-spells the command path.
//
// P0 is Python-only and takes no language selector: the verb is bare
// "setup-local" (spec §naming). A language axis (setup-local python / scala) is
// the preferred shape only if more languages are ever added, and nothing is
// reserved for it here.
const (
	CommandGroup = "environments"
	CommandVerb  = "setup-local"
	CommandName  = CommandGroup + " " + CommandVerb

	// SchemaVersion is the version of the --json output contract (spec §6).
	// Bump it on any breaking change to the JSON shape.
	SchemaVersion = 1
)

// Mode is the provisioning mode: a full environment (default) or the
// constraints-only variant that omits the databricks-connect dependency.
type Mode int

const (
	ModeDefault Mode = iota
	ModeConstraintsOnly
)

// String returns the JSON/text spelling of the mode ("default" | "constraints-only").
func (m Mode) String() string {
	if m == ModeConstraintsOnly {
		return "constraints-only"
	}
	return "default"
}

// PhaseName is a canonical execution phase (spec §3 / §6). The set is fixed and
// ordered; the --json "phases" array reports every phase in this order.
type PhaseName string

const (
	PhasePreflight PhaseName = "preflight"
	PhaseResolve   PhaseName = "resolve"
	PhaseFetch     PhaseName = "fetch"
	PhaseMerge     PhaseName = "merge"
	PhaseProvision PhaseName = "provision"
	PhaseValidate  PhaseName = "validate"
)

// Reporter receives phase-start notifications during a run so a caller can show
// live progress (e.g. a spinner). It is intentionally minimal: the pipeline owns
// success/failure reporting via the Result, and Reporter only marks entry into a
// phase. A nil Reporter disables progress.
type Reporter interface {
	PhaseStarted(name PhaseName)
}

// Phase status values (spec §6.2).
const (
	StatusOK      = "ok"
	StatusError   = "error"
	StatusPending = "pending"
)

// ErrorCode is a stable failure-class identifier surfaced in --json error.code
// (spec §7). Values are compared via the ErrorCode constants, never by
// string-matching messages, and are defined once here.
//
// This set is the source of truth for the spec's error-code table; each code is
// annotated with the phase that emits it. Two codes from the spec are
// intentionally not defined because the CLI never emits them: E_PYTHON_POLICY
// (a policy-gated failure whose signal source does not yet exist) and E_AUTH
// (the shared workspace-client preflight, MustWorkspaceClient, surfaces the
// standard CLI auth error before a command JSON object is ever built).
type ErrorCode string

const (
	ErrUsage              ErrorCode = "E_USAGE"               // preflight: incompatible flags; resolve: --job-task names a job but no task
	ErrManagerUnsupported ErrorCode = "E_MANAGER_UNSUPPORTED" // preflight: manager is not uv
	ErrNotWritable        ErrorCode = "E_NOT_WRITABLE"        // preflight: project dir not writable
	ErrUvMissing          ErrorCode = "E_UV_MISSING"          // preflight: uv not found / install failed
	ErrNoTarget           ErrorCode = "E_NO_TARGET"           // resolve: no target from any source
	ErrResolve            ErrorCode = "E_RESOLVE"             // resolve: target read failed / ambiguous name
	ErrEnvUnsupported     ErrorCode = "E_ENV_UNSUPPORTED"     // fetch: no published env key
	ErrFetch              ErrorCode = "E_FETCH"               // fetch: repo unreachable, no usable cache
	ErrWrite              ErrorCode = "E_WRITE"               // merge: greenfield write failed
	ErrMerge              ErrorCode = "E_MERGE"               // merge: existing-project merge failed
	ErrPythonInstall      ErrorCode = "E_PYTHON_INSTALL"      // provision: uv python install failed
	ErrProvision          ErrorCode = "E_PROVISION"           // provision: uv sync failed
	ErrValidate           ErrorCode = "E_VALIDATE"            // validate: post-provision version mismatch

	// ErrCanceled is not in the spec's error-code table: it reports a user/parent
	// interrupt (SIGINT/SIGTERM cancels the context), not a failure of the phase
	// it happened to be in. Without it an interrupt mid-`uv sync` surfaces as
	// E_PROVISION with a "provision failed" message, implying something broke when
	// the user simply pressed Ctrl-C. FailurePhase still records where it stopped.
	ErrCanceled ErrorCode = "E_CANCELED" // any phase: interrupted by SIGINT/SIGTERM
)

// PipelineError is a failure carrying a stable code, the phase at which it
// occurred, and whether disk was mutated before the failure. It marshals to the
// --json error object (spec §6.2). Code and FailurePhase are the stable
// contract; Err holds the wrapped cause for errors.Is/As and is not serialized
// directly (its text is folded into the "message" field — see MarshalJSON).
type PipelineError struct {
	Code         ErrorCode `json:"code"`
	FailurePhase PhaseName `json:"failurePhase"`
	Msg          string    `json:"message"`
	DiskMutated  bool      `json:"diskMutated"`
	Err          error     `json:"-"`
}

func (e *PipelineError) Error() string {
	if e.Err != nil {
		return e.Msg + ": " + e.Err.Error()
	}
	return e.Msg
}

func (e *PipelineError) Unwrap() error {
	return e.Err
}

// MarshalJSON serializes the full message (Error(), i.e. Msg plus any wrapped
// cause) into the "message" field. Without this the --json error object would
// carry only Msg and drop the cause (Err is json:"-"), so a JSON consumer would
// get strictly less detail than the text output — e.g. "resolving cluster name"
// without the "ambiguous"/"not found" reason. Text and JSON must agree.
func (e *PipelineError) MarshalJSON() ([]byte, error) {
	type alias PipelineError // avoid recursing into this method
	return json.Marshal((*alias)(&PipelineError{
		Code:         e.Code,
		FailurePhase: e.FailurePhase,
		Msg:          e.Error(),
		DiskMutated:  e.DiskMutated,
	}))
}

// NewError creates a PipelineError with a code and message. FailurePhase and
// DiskMutated are filled in by the pipeline when it records the failure. The
// message is formatted with fmt.Sprintf(format, args...); err may be nil.
func NewError(code ErrorCode, err error, format string, args ...any) *PipelineError {
	return &PipelineError{
		Code: code,
		Msg:  fmt.Sprintf(format, args...),
		Err:  err,
	}
}

// ComputeInfo is the resolved compute target, serialized as the result's
// "compute" key (spec §6). Source records which precedence source was used
// ("cluster", "serverless", "job", or "bundle"). SparkVersion is the raw cluster
// runtime string the resolver read;
// it is folded into EnvKey (dbr/<SparkVersion>) and is not part of the JSON
// contract, kept only as intermediate resolver state.
type ComputeInfo struct {
	Source            string `json:"source"`
	ClusterID         string `json:"clusterId,omitempty"`
	ServerlessVersion string `json:"serverlessVersion,omitempty"`
	EnvKey            string `json:"envKey"`

	SparkVersion string `json:"-"`
}

// Label returns a short, human-readable name for the resolved compute target,
// for display in the text summary (e.g. "serverless 4", "cluster 0101-abc",
// "DBR 15.4.x-scala2.12"). The precise environment key is still available in
// --output json and --debug.
func (c *ComputeInfo) Label() string {
	switch {
	case c.ServerlessVersion != "":
		// ServerlessVersion is normalized to "v4"; drop the "v" for display.
		return "serverless " + c.ServerlessEnvironmentVersion()
	case c.ClusterID != "":
		return "cluster " + c.ClusterID
	case c.SparkVersion != "":
		// Classic compute resolved from a --job-task carries no ClusterID (only the
		// task's runtime), so without this case Label would fall through and print
		// the internal "dbr/..." env key — exactly the detail the summary hides.
		// Show the runtime instead, matching the "DBR <version>" phrasing used in help.
		return "DBR " + c.SparkVersion
	default:
		return c.EnvKey
	}
}

// ServerlessEnvironmentVersion returns the bare serverless environment version
// (e.g. "5") to write into [tool.databricks.environment].environment_version.
// ServerlessVersion is normalized to "vN", so the leading "v" is dropped to
// match the documented bare-number form. It is empty for a cluster target
// (which leaves ServerlessVersion unset), where the section is not managed.
func (c *ComputeInfo) ServerlessEnvironmentVersion() string {
	return strings.TrimPrefix(c.ServerlessVersion, "v")
}

// ResolvedInfo is the resolved environment definition (spec §6 "resolved").
// DBConnectVersion is omitted in constraints-only mode.
type ResolvedInfo struct {
	PythonVersion    string `json:"pythonVersion"`
	DBConnectVersion string `json:"dbconnectVersion,omitempty"`
	ArtifactSource   string `json:"artifactSource"`
}

// Plan describes the changes a --dry-run run would apply (spec §6.3).
// ChangedRegions is retained for text output only and is not serialized.
type Plan struct {
	WouldWrite         string `json:"wouldWrite"`
	WouldBackup        string `json:"wouldBackup,omitempty"`
	WouldInstallPython string `json:"wouldInstallPython,omitempty"`
	Diff               string `json:"diff"`

	ChangedRegions []string `json:"-"`
}

// PhaseStatus is one entry in the --json "phases" array (spec §6). Detail is
// used for human-readable text output only and is not serialized.
type PhaseStatus struct {
	Phase  PhaseName `json:"phase"`
	Status string    `json:"status"`

	Detail string `json:"-"`
}

// Warning is a non-fatal advisory surfaced in --json "warnings" (spec §6). Code
// is a stable, categorical identifier from the closed set below; Message is
// human-readable text for the text renderer and is not part of the contract.
type Warning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Warning codes are the closed, categorical set surfaced in --json warnings[].
// They let a consumer report a count and a code histogram (merge quality) without
// parsing free-form text. All are emitted from the merge phase, where the fetched
// env-owned pins can conflict with what the user already had.
//
// The messages are phrased in the present tense because the same detection runs
// under --dry-run, where nothing has been written yet.
const (
	// WarnRequiresPythonOverridden: the user's [project].requires-python differs
	// from the env's pin and is replaced by the managed value.
	WarnRequiresPythonOverridden = "W_REQUIRES_PYTHON_OVERRIDDEN"
	// WarnDBConnectPinOverridden: the user's databricks-connect pin sits directly in
	// the dev group and is replaced by the managed value.
	WarnDBConnectPinOverridden = "W_DBCONNECT_PIN_OVERRIDDEN"
	// WarnDBConnectConsolidated: a databricks-connect pin the user had outside the
	// managed dev entry — in [project].dependencies, an optional-dependency extra, or
	// another dependency group — was disjoint from the environment's version and is
	// removed, so what remains resolves. Only conflicting pins are removed; a pin that
	// co-resolves, carries no version, or is marker-gated is left in place. Emitted once
	// per removed pin. Informational: the merge makes the project resolvable.
	WarnDBConnectConsolidated = "W_DBCONNECT_CONSOLIDATED"
	// WarnDBConnectPinDuplicated: a databricks-connect pin of the user's is one the
	// merge can neither rewrite nor remove — a spelling the line-based passes do not
	// reach (a single-quoted element, a pin under a quoted TOML key, or an inline-table
	// or dotted sub-table form) — so the managed pin lands in the dev group alongside
	// it. Unlike an override this leaves two pins for one package and, where their
	// ranges are disjoint, uv cannot resolve it — a distinct and worse outcome that
	// needs a manual fix, so it carries its own code. It can accompany an override or a
	// consolidation and persists across re-runs for as long as the survivor does.
	WarnDBConnectPinDuplicated = "W_DBCONNECT_PIN_DUPLICATED"
	// WarnUserConstraintConflict: a user dependency pins a package that the env's
	// constraint-dependencies also constrains, to a provably non-overlapping version
	// range (uv will fail to resolve). Every requirement uv locks is scanned —
	// [project].dependencies, the optional-dependency extras, and all dependency
	// groups — since constraint-dependencies applies to the whole resolution. Emitted
	// only when the ranges are provably disjoint; ambiguous cases are not flagged.
	WarnUserConstraintConflict = "W_USER_CONSTRAINT_CONFLICT"
	// WarnStaleEnvironmentVersion: the target is a cluster, which does not manage the
	// serverless environment section, but the file carries a [tool.databricks.environment]
	// environment_version left over from an earlier serverless run. The value is not
	// updated (cluster targets are a no-op there), so it now describes a target the
	// project is no longer set up for — worth surfacing because VS Code and serverless
	// Jobs read that section as a source of truth.
	WarnStaleEnvironmentVersion = "W_STALE_ENVIRONMENT_VERSION"
	// WarnStandalonePysparkConflict: the user declares a standalone pyspark dependency
	// while the environment installs databricks-connect. databricks-connect vendors its
	// own pyspark (it ships the pyspark/ package tree rather than depending on the
	// standalone distribution), so a separately declared pyspark resolves into the same
	// namespace and the two overwrite each other — the environment then fails to start a
	// session. This is a coexistence conflict, not a version one, so it is reported
	// independent of the pyspark version pinned. Emitted whenever databricks-connect
	// ends up in the resolved environment — whether the env manages it (default mode) or
	// the user's own pyproject pins it (constraints-only mode) — so it agrees with the
	// validate hard-fail, which keys on the installed venv rather than the mode.
	WarnStandalonePysparkConflict = "W_STANDALONE_PYSPARK_CONFLICT"
)

// Result is the full outcome of a sync run and the root of the --json object
// (spec §6). Field order matches the spec's schema so JSON key order is stable.
//
// Phases and Warnings are non-omitempty slices, so they must always be non-nil
// before marshalling or the --json contract would emit "null" instead of "[]" —
// a distinction that trips JSON consumers and golden diffs. Construct a Result
// with NewResult (or otherwise seed both) rather than a bare Result{} literal.
type Result struct {
	SchemaVersion int            `json:"schemaVersion"`
	Command       string         `json:"command"`
	OK            bool           `json:"ok"`
	Mode          string         `json:"mode"`
	DryRun        bool           `json:"dryRun"`
	Compute       *ComputeInfo   `json:"compute,omitempty"`
	Resolved      *ResolvedInfo  `json:"resolved,omitempty"`
	Greenfield    bool           `json:"greenfield"`
	Plan          *Plan          `json:"plan,omitempty"`
	VenvPath      string         `json:"venvPath,omitempty"`
	Phases        []PhaseStatus  `json:"phases"`
	Warnings      []Warning      `json:"warnings"`
	Error         *PipelineError `json:"error"`
	BackupPath    string         `json:"backupPath,omitempty"`
	// DurationMs is the pipeline's wall time in milliseconds (spec §6). It covers the
	// CLI pipeline only; the extension measures its own end-to-end latency (process
	// spawn, interpreter adoption) separately.
	DurationMs int64 `json:"durationMs"`
}

// NewResult returns a Result with the non-omitempty slice fields initialized to
// empty (non-nil) slices, so the --json output always renders "phases": [] and
// "warnings": [] rather than "null". Callers fill in the remaining fields.
func NewResult() *Result {
	return &Result{
		Phases:   []PhaseStatus{},
		Warnings: []Warning{},
	}
}
