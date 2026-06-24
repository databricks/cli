package dbconnect

import "fmt"

// Mode represents the dbconnect operation mode.
type Mode int

const (
	ModeInit Mode = iota
	ModeSync
)

// String returns the string representation of the Mode.
func (m Mode) String() string {
	if m == ModeInit {
		return "init"
	}
	return "sync"
}

// ErrorCode represents a dbconnect error code.
type ErrorCode string

const (
	ErrNoTargetSelected      ErrorCode = "no_target_selected"
	ErrConstraintFetchFailed ErrorCode = "constraint_fetch_failed"
	ErrMergeFailed           ErrorCode = "merge_failed"
	ErrProvisionFailed       ErrorCode = "provision_failed"
	ErrValidationFailed      ErrorCode = "validation_failed"
	ErrUvUnavailable         ErrorCode = "uv_unavailable"
)

// PipelineError represents an error during the dbconnect pipeline.
type PipelineError struct {
	Code ErrorCode `json:"code"`
	Msg  string    `json:"message"`
	Err  error     `json:"-"`
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

// NewError creates a new PipelineError. The message is formatted using fmt.Sprintf(format, args...),
// and err may be nil.
func NewError(code ErrorCode, err error, format string, args ...any) *PipelineError {
	return &PipelineError{
		Code: code,
		Msg:  fmt.Sprintf(format, args...),
		Err:  err,
	}
}

// TargetInfo contains information about the target environment.
type TargetInfo struct {
	Kind          string `json:"kind"`
	ClusterID     string `json:"cluster_id"`
	SparkVersion  string `json:"spark_version"`
	EnvKey        string `json:"env_key"`
	PythonVersion string `json:"python_version"`
}

// ConstraintInfo contains constraint information.
type ConstraintInfo struct {
	SourceURL         string `json:"source_url"`
	FromCache         bool   `json:"from_cache"`
	RequiresPython    string `json:"requires_python"`
	DatabricksConnect string `json:"databricks_connect"`
	ConstraintCount   int    `json:"constraint_count"`
}

// Plan contains the deployment plan.
type Plan struct {
	PyprojectPath  string   `json:"pyproject_path"`
	BackupPath     string   `json:"backup_path"`
	Diff           string   `json:"diff"`
	ChangedRegions []string `json:"changed_regions"`
}

// PhaseResult contains the result of a single phase.
type PhaseResult struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

// ResultDetail contains the final result details.
type ResultDetail struct {
	Status                     string `json:"status"`
	VenvPath                   string `json:"venv_path"`
	PythonVersion              string `json:"python_version"`
	DatabricksConnectInstalled string `json:"databricks_connect_installed"`
}

// Result contains the overall result of the dbconnect operation.
type Result struct {
	Mode        string          `json:"mode"`
	Check       bool            `json:"check"`
	Target      *TargetInfo     `json:"target,omitempty"`
	Constraints *ConstraintInfo `json:"constraints,omitempty"`
	Plan        *Plan           `json:"plan,omitempty"`
	Phases      []PhaseResult   `json:"phases,omitempty"`
	Result      *ResultDetail   `json:"result,omitempty"`
	Error       *PipelineError  `json:"error,omitempty"`
}
