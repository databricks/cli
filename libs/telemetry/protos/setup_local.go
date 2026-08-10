package protos

// SetupLocalComputeSource mirrors SetupLocalComputeSource.Type in the
// databricks_cli lumberjack proto. Unspecified absorbs a source a newer CLI
// knows about but the server proto does not yet.
type SetupLocalComputeSource string

const (
	SetupLocalComputeSourceUnspecified SetupLocalComputeSource = "TYPE_UNSPECIFIED"
	SetupLocalComputeSourceCluster     SetupLocalComputeSource = "CLUSTER"
	SetupLocalComputeSourceServerless  SetupLocalComputeSource = "SERVERLESS"
	SetupLocalComputeSourceJob         SetupLocalComputeSource = "JOB"
	SetupLocalComputeSourceBundle      SetupLocalComputeSource = "BUNDLE"
)

// SetupLocalMode mirrors SetupLocalMode.Type in the databricks_cli lumberjack proto.
type SetupLocalMode string

const (
	SetupLocalModeUnspecified     SetupLocalMode = "TYPE_UNSPECIFIED"
	SetupLocalModeDefault         SetupLocalMode = "DEFAULT"
	SetupLocalModeConstraintsOnly SetupLocalMode = "CONSTRAINTS_ONLY"
)

// SetupLocalErrorCode mirrors SetupLocalErrorCode.Type in the databricks_cli
// lumberjack proto. It carries the pipeline's stable failure class on a failed
// run and is empty on success.
type SetupLocalErrorCode string

const (
	SetupLocalErrorCodeUnspecified        SetupLocalErrorCode = "TYPE_UNSPECIFIED"
	SetupLocalErrorCodeUsage              SetupLocalErrorCode = "E_USAGE"
	SetupLocalErrorCodeManagerUnsupported SetupLocalErrorCode = "E_MANAGER_UNSUPPORTED"
	SetupLocalErrorCodeNotWritable        SetupLocalErrorCode = "E_NOT_WRITABLE"
	SetupLocalErrorCodeUvMissing          SetupLocalErrorCode = "E_UV_MISSING"
	SetupLocalErrorCodeNoTarget           SetupLocalErrorCode = "E_NO_TARGET"
	SetupLocalErrorCodeResolve            SetupLocalErrorCode = "E_RESOLVE"
	SetupLocalErrorCodeEnvUnsupported     SetupLocalErrorCode = "E_ENV_UNSUPPORTED"
	SetupLocalErrorCodeFetch              SetupLocalErrorCode = "E_FETCH"
	SetupLocalErrorCodeWrite              SetupLocalErrorCode = "E_WRITE"
	SetupLocalErrorCodeMerge              SetupLocalErrorCode = "E_MERGE"
	SetupLocalErrorCodePythonInstall      SetupLocalErrorCode = "E_PYTHON_INSTALL"
	SetupLocalErrorCodeProvision          SetupLocalErrorCode = "E_PROVISION"
	SetupLocalErrorCodeValidate           SetupLocalErrorCode = "E_VALIDATE"
	SetupLocalErrorCodeCanceled           SetupLocalErrorCode = "E_CANCELED"
)

// SetupLocalPhase mirrors SetupLocalPhase.Type in the databricks_cli lumberjack
// proto. On a failed run it records which phase failed; empty on success.
type SetupLocalPhase string

const (
	SetupLocalPhaseUnspecified SetupLocalPhase = "TYPE_UNSPECIFIED"
	SetupLocalPhasePreflight   SetupLocalPhase = "preflight"
	SetupLocalPhaseResolve     SetupLocalPhase = "resolve"
	SetupLocalPhaseFetch       SetupLocalPhase = "fetch"
	SetupLocalPhaseMerge       SetupLocalPhase = "merge"
	SetupLocalPhaseProvision   SetupLocalPhase = "provision"
	SetupLocalPhaseValidate    SetupLocalPhase = "validate"
)

// SetupLocalEvent is emitted on every real (non-dry-run) execution of the
// `databricks environments setup-local` command.
type SetupLocalEvent struct {
	// Whether the run completed successfully. Not omitempty: a genuine false
	// must be distinguishable from an older CLI that did not report the field.
	Success bool `json:"success"`

	// Which precedence source supplied the compute target.
	ComputeSource SetupLocalComputeSource `json:"compute_source,omitempty"`

	// Provisioning mode: default or constraints-only.
	Mode SetupLocalMode `json:"mode,omitempty"`

	// Whether pyproject.toml was created fresh (true) vs an existing project
	// updated (false). Not omitempty for the same reason as Success.
	Greenfield bool `json:"greenfield"`

	// Published constraint catalog key, e.g. "dbr/16.4" or
	// "serverless/serverless-v5". Not PII.
	EnvKey string `json:"env_key,omitempty"`

	// Stable failure class on a failed run; empty on success.
	ErrorCode SetupLocalErrorCode `json:"error_code,omitempty"`

	// Phase that failed on a failed run; empty on success.
	FailurePhase SetupLocalPhase `json:"failure_phase,omitempty"`

	// Whether a failure left the project's pyproject.toml partially written.
	// Not omitempty so false is reported explicitly.
	DiskMutated bool `json:"disk_mutated"`
}
