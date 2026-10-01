package protos

type AirGPUType string

const (
	AirGPUTypeUnspecified AirGPUType = "GPU_TYPE_UNSPECIFIED"
	AirGPUType1xA10       AirGPUType = "GPU_1X_A10"
	AirGPUType1xH100      AirGPUType = "GPU_1X_H100"
	AirGPUType8xH100      AirGPUType = "GPU_8X_H100"
	AirGPUType8xB300      AirGPUType = "GPU_8X_B300"
)

type AirPackagingMode string

const (
	AirPackagingModeGitArchive AirPackagingMode = "GIT_ARCHIVE"
	AirPackagingModePlainTar   AirPackagingMode = "PLAIN_TAR"
)

// AirRunEvent describes an AIR submission, matching the workload measurements
// collected by the Python AIR CLI. It contains no user-authored names or paths.
type AirRunEvent struct {
	GPUType               AirGPUType `json:"gpu_type"`
	NumGPUs               int        `json:"num_gpus"`
	NumNodes              int        `json:"num_nodes"`
	HasDockerImage        bool       `json:"has_docker_image"`
	HasCodeSnapshot       bool       `json:"has_code_snapshot"`
	HasRequirements       bool       `json:"has_requirements"`
	HasParameters         bool       `json:"has_parameters"`
	MaxRetries            int        `json:"max_retries"`
	HasTimeout            bool       `json:"has_timeout"`
	SubmittedSuccessfully bool       `json:"submitted_successfully"`
	SubmitLatencyMs       int64      `json:"submit_latency_ms"`
	JobRunID              string     `json:"job_run_id,omitempty"`
	CodeSourceUsesGit     *bool      `json:"code_source_uses_git,omitempty"`

	// Compressed tarball bytes, including when reusing a cached snapshot.
	// Absent when no archive was measured; a measured zero remains explicit.
	CodeSourceSizeBytes *int64 `json:"code_source_size_bytes,omitempty"`

	CodeSourcePackagingMode AirPackagingMode `json:"code_source_packaging_mode,omitempty"`
	// Wall-clock milliseconds for archive creation and the tarball write,
	// respectively. Zero on cache hits; absent if the phase was never attempted.
	CodeSourcePackagingDurationMs *int64 `json:"code_source_packaging_duration_ms,omitempty"`
	CodeSourceUploadDurationMs    *int64 `json:"code_source_upload_duration_ms,omitempty"`
}
