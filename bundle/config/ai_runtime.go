package config

// AiRuntimeTaskExtras holds the DABs-native, non-proto authoring sugar for one
// ai_runtime_task. These are not fields on the SDK jobs.AiRuntimeTask: they are
// extracted from the config tree by rewriteAiRuntimeCodeSource before normalization
// (which drops unknown struct fields) and lowered into the task by the aicode
// build-phase mutator. Today only code_source; parameters (lowered to a
// hyperparameters.yaml sidecar) is the intended next field.
type AiRuntimeTaskExtras struct {
	CodeSource *CodeSourceOptions `json:"code_source,omitempty"`
}

// CodeSourceOptions is the DABs-native `ai_runtime_task.code_source` block. It gives
// train.yaml parity for code delivery: at deploy the aicode mutator packages the
// directory (honoring .gitignore + sync include/exclude, plus git ref and
// include_paths when set), uploads it, and sets the real SDK field
// ai_runtime_task.code_source_path to the uploaded path.
type CodeSourceOptions struct {
	// RootPath is the local directory to package, relative to the bundle sync root.
	RootPath string `json:"root_path"`

	// IncludePaths narrows the archive to these subtrees of RootPath (relative, no
	// "..") instead of packaging the whole directory.
	IncludePaths []string `json:"include_paths,omitempty"`

	// Git pins the archive to a committed revision instead of the working tree.
	Git *CodeSourceGit `json:"git,omitempty"`

	// RemoteVolume uploads the archive to this UC Volume path (starts with
	// "/Volumes/") instead of the bundle's workspace file path.
	RemoteVolume string `json:"remote_volume,omitempty"`
}

// CodeSourceGit pins a code_source snapshot to a git ref. Branch and commit are
// mutually exclusive.
type CodeSourceGit struct {
	Branch string `json:"branch,omitempty"`
	Commit string `json:"commit,omitempty"`
}
