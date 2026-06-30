package aircmd

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"
)

// This file ports the run YAML schema and its structural validation from the
// Python CLI's sdk/config.py. "Structural" means types, required fields, and
// format/cross-field rules that need no workspace access. Online checks (e.g.
// GPU availability) and git/filesystem checks run at launch time and are
// intentionally not ported here.
//
// Divergences from the Python schema: compute.node_pool_id / compute.pool_name
// (see compute.go) and the top-level `priority` field are dropped because AIR
// does not support node-pool placement. priority is a pool-queue-ordering knob,
// so it goes with the pool fields.

// REGEX_TASK_KEY_CHARS: ASCII alphanumeric, hyphen, underscore only (no periods).
// Explicit ASCII class, not \w: \w matches Unicode letters that the ASCII-only
// Jobs API task_key rejects.
var taskKeyRe = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

// gitRefRe guards branch/remote names against command injection. Only safe ref
// characters are allowed.
var gitRefRe = regexp.MustCompile(`^[\w./-]+$`)

// runConfig is the top-level run YAML schema: experiment_name + compute /
// environment / code_source plus the command and run options.
type runConfig struct {
	ExperimentName string             `yaml:"experiment_name"`
	Compute        *computeConfig     `yaml:"compute"`
	Environment    *environmentConfig `yaml:"environment"`
	Command        *string            `yaml:"command"`
	EnvVariables   map[string]string  `yaml:"env_variables"`
	Secrets        map[string]string  `yaml:"secrets"`
	CodeSource     *codeSourceConfig  `yaml:"code_source"`
	// MaxRetries defaults to 3 when unset; default-filling is a normalization
	// concern handled at launch, so a nil pointer is left as-is here.
	MaxRetries                *int           `yaml:"max_retries"`
	TimeoutMinutes            *int           `yaml:"timeout_minutes"`
	IdempotencyToken          *string        `yaml:"idempotency_token"`
	Parameters                map[string]any `yaml:"parameters"`
	MLflowRunName             *string        `yaml:"mlflow_run_name"`
	MLflowExperimentDirectory *string        `yaml:"mlflow_experiment_directory"`
	Permissions               []permission   `yaml:"permissions"`
	UsagePolicyName           *string        `yaml:"usage_policy_name"`
	UsagePolicyID             *string        `yaml:"usage_policy_id"`
}

// validate runs structural validation over the whole config, returning the first
// failure. Fields are checked in declaration order to keep error output stable.
func (c *runConfig) validate() error {
	if err := validateExperimentName(c.ExperimentName); err != nil {
		return err
	}

	if c.Compute == nil {
		return errors.New("compute: section is required")
	}
	if err := c.Compute.validate(); err != nil {
		return err
	}

	if c.Environment != nil {
		if err := c.Environment.validate(); err != nil {
			return err
		}
	}

	// command is optional in the type system but required in practice, matching
	// the Python validate_script_fields model validator.
	if c.Command == nil {
		return errors.New("command is required")
	}
	if err := validateCommand(*c.Command); err != nil {
		return err
	}

	if err := validateSecretRefs(c.Secrets); err != nil {
		return err
	}

	// A name can't be both a plain env var and a secret: the precedence would be
	// ambiguous and could leak the secret. Sorted for a stable error.
	for _, name := range slices.Sorted(maps.Keys(c.EnvVariables)) {
		if _, ok := c.Secrets[name]; ok {
			return fmt.Errorf("%q is set in both env_variables and secrets; remove it from one", name)
		}
	}

	if c.CodeSource != nil {
		if err := c.CodeSource.validate(); err != nil {
			return err
		}
	}

	if c.MaxRetries != nil && *c.MaxRetries < 0 {
		return fmt.Errorf("max_retries must be >= 0, got %d", *c.MaxRetries)
	}

	if c.TimeoutMinutes != nil && *c.TimeoutMinutes < 1 {
		return fmt.Errorf("timeout_minutes must be >= 1, got %d", *c.TimeoutMinutes)
	}

	if c.IdempotencyToken != nil {
		v := strings.TrimSpace(*c.IdempotencyToken)
		if v == "" {
			return errors.New("idempotency_token cannot be empty")
		}
		if len(v) > 64 {
			return errors.New("idempotency_token must be 64 characters or less")
		}
	}

	if c.MLflowRunName != nil {
		v := strings.TrimSpace(*c.MLflowRunName)
		if v == "" {
			return errors.New("mlflow_run_name cannot be empty")
		}
		if len(v) > 100 {
			return fmt.Errorf("mlflow_run_name must be 100 characters or less (got %d)", len(v))
		}
		if !taskKeyRe.MatchString(v) {
			return fmt.Errorf("invalid mlflow_run_name %q: only alphanumeric characters, hyphens, and underscores are allowed", v)
		}
	}

	if c.MLflowExperimentDirectory != nil {
		v := strings.TrimSpace(*c.MLflowExperimentDirectory)
		if v == "" {
			return errors.New("mlflow_experiment_directory cannot be empty")
		}
		// MLflow experiments live under the workspace tree.
		if !strings.HasPrefix(v, "/Workspace") {
			return fmt.Errorf("mlflow_experiment_directory must start with '/Workspace', got: %s", v)
		}
	}

	for i := range c.Permissions {
		if err := c.Permissions[i].validate(); err != nil {
			return err
		}
	}

	// A usage policy is given by name or id, never both; the name resolves to an
	// id at launch.
	if c.UsagePolicyName != nil && c.UsagePolicyID != nil {
		return errors.New("usage_policy_name and usage_policy_id are mutually exclusive; set only one")
	}
	if c.UsagePolicyName != nil {
		v := strings.TrimSpace(*c.UsagePolicyName)
		if v == "" {
			return errors.New("usage_policy_name must not be empty")
		}
		// 127 matches the server-side max_length on the policy name filter.
		if len(v) > 127 {
			return fmt.Errorf("usage_policy_name must be at most 127 characters, got %d", len(v))
		}
	}
	if c.UsagePolicyID != nil && strings.TrimSpace(*c.UsagePolicyID) == "" {
		return errors.New("usage_policy_id must not be empty")
	}

	return nil
}

// validateExperimentName enforces the Databricks Jobs API task_key constraints:
// the experiment_name becomes a task key, which caps at 100 characters and allows
// only alphanumerics, hyphens, and underscores.
func validateExperimentName(v string) error {
	if v == "" {
		return errors.New("experiment_name cannot be empty")
	}
	if len(v) > 100 {
		return fmt.Errorf("experiment_name must be 100 characters or less (got %d); this is the Jobs API task_key length limit", len(v))
	}
	if !taskKeyRe.MatchString(v) {
		return fmt.Errorf("invalid experiment_name %q: only alphanumeric characters, hyphens (-), and underscores (_) are allowed", v)
	}
	return nil
}

// validateCommand enforces command is non-empty and within the line-count cap.
func validateCommand(v string) error {
	if strings.TrimSpace(v) == "" {
		return errors.New("command cannot be empty")
	}
	lineCount := strings.Count(v, "\n") + 1
	if lineCount > 1000 {
		return fmt.Errorf("command is too long (%d lines); maximum is 1000 lines — move complex logic into a script in your code_source", lineCount)
	}
	return nil
}

// validateSecretRefs checks that secret references use the "scope/key" format.
func validateSecretRefs(secrets map[string]string) error {
	for varName, ref := range secrets {
		parts := strings.Split(ref, "/")
		if len(parts) != 2 {
			return fmt.Errorf("invalid secret reference %q for variable %q: expected format 'scope/key' (e.g., my_scope/hf_token)", ref, varName)
		}
		if parts[0] == "" || parts[1] == "" {
			return fmt.Errorf("invalid secret reference %q for variable %q: scope and key cannot be empty", ref, varName)
		}
	}
	return nil
}

// environmentConfig is the `environment` block: dependencies and/or a custom
// docker image.
type environmentConfig struct {
	Dependencies dependencies       `yaml:"dependencies"`
	Version      stringOrInt        `yaml:"version"`
	DockerImage  *dockerImageConfig `yaml:"docker_image"`
}

func (e *environmentConfig) validate() error {
	// docker_image is exclusive with dependencies/version: the image already pins
	// the full runtime.
	if e.DockerImage != nil {
		var conflicting []string
		if e.Dependencies.set {
			conflicting = append(conflicting, "dependencies")
		}
		if e.Version.set {
			conflicting = append(conflicting, "version")
		}
		if len(conflicting) > 0 {
			return fmt.Errorf("when 'docker_image' is specified under 'environment', these fields are not allowed: %s", strings.Join(conflicting, ", "))
		}
		return e.DockerImage.validate()
	}

	// version pins the client image version, which is only meaningful for an
	// inline (list) dependency set — a requirements.yaml file carries its own.
	if e.Version.set {
		if e.Dependencies.set && !e.Dependencies.isList {
			return errors.New("'environment.version' is only valid with inline dependencies (a list); when 'dependencies' points to a requirements.yaml file, set the version inside that file")
		}
		if !e.Dependencies.set {
			return errors.New("'environment.version' requires inline 'dependencies' (a list of packages)")
		}
	}

	return nil
}

// dependencies is environment.dependencies, which is polymorphic: a string is a
// path to a requirements.yaml file; a list is an inline package list.
type dependencies struct {
	set    bool
	isList bool
	path   string
	list   []string
}

func (d *dependencies) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		d.set, d.isList = true, false
		return node.Decode(&d.path)
	case yaml.SequenceNode:
		d.set, d.isList = true, true
		return node.Decode(&d.list)
	default:
		return errors.New("environment.dependencies must be a string path or a list of packages")
	}
}

// stringOrInt holds a scalar that may be a string or an integer in YAML
// (environment.version). The raw text is kept; integer-format validation is a
// launch-time concern.
type stringOrInt struct {
	set bool
	raw string
}

func (s *stringOrInt) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return errors.New("environment.version must be a string or integer")
	}
	s.set = true
	s.raw = node.Value
	return nil
}

// dockerImageConfig is environment.docker_image.
type dockerImageConfig struct {
	URL string `yaml:"url"`
}

func (d *dockerImageConfig) validate() error {
	if strings.TrimSpace(d.URL) == "" {
		return errors.New("docker_image.url cannot be empty")
	}
	return nil
}

// codeSourceConfig is the `code_source` block. Only the "snapshot" type exists.
type codeSourceConfig struct {
	Type     string                `yaml:"type"`
	Snapshot *snapshotSourceConfig `yaml:"snapshot"`
}

func (c *codeSourceConfig) validate() error {
	if c.Type != "snapshot" {
		return fmt.Errorf("code_source.type must be 'snapshot', got %q", c.Type)
	}
	if c.Snapshot == nil {
		return errors.New("code_source.type='snapshot' requires a snapshot configuration")
	}
	return c.Snapshot.validate()
}

// snapshotSourceConfig describes a local directory to tar and upload.
type snapshotSourceConfig struct {
	RootPath     string   `yaml:"root_path"`
	RemoteVolume *string  `yaml:"remote_volume"`
	Git          *gitRef  `yaml:"git"`
	IncludePaths []string `yaml:"include_paths"`
}

func (s *snapshotSourceConfig) validate() error {
	if strings.TrimSpace(s.RootPath) == "" {
		return errors.New("code_source.snapshot.root_path cannot be empty")
	}

	if s.RemoteVolume != nil && !strings.HasPrefix(*s.RemoteVolume, "/Volumes/") {
		return errors.New("code_source.snapshot.remote_volume must start with '/Volumes/'")
	}

	// A non-nil but empty include_paths is an explicit mistake (omit it instead).
	if s.IncludePaths != nil && len(s.IncludePaths) == 0 {
		return errors.New("code_source.snapshot.include_paths cannot be an empty list; either omit it or provide paths")
	}
	for _, p := range s.IncludePaths {
		p = strings.TrimSpace(p)
		if p == "" {
			return errors.New("code_source.snapshot.include_paths entry cannot be empty")
		}
		if strings.HasPrefix(p, "/") {
			return fmt.Errorf("code_source.snapshot.include_paths must be relative paths, got: %s", p)
		}
		// No parent traversal: snapshots must stay within root_path.
		if slices.Contains(strings.Split(p, "/"), "..") {
			return fmt.Errorf("code_source.snapshot.include_paths cannot contain '..' traversal, got: %s", p)
		}
	}

	if s.Git != nil {
		return s.Git.validate()
	}
	return nil
}

// gitRef pins a snapshot to a specific git ref. branch and commit are mutually
// exclusive; remote is only meaningful with branch.
type gitRef struct {
	Branch *string   `yaml:"branch"`
	Commit *string   `yaml:"commit"`
	Remote gitRemote `yaml:"remote"`
}

func (g *gitRef) validate() error {
	if g.Branch != nil && !gitRefRe.MatchString(*g.Branch) {
		return fmt.Errorf("invalid git.branch format %q: only alphanumeric characters, hyphens, dots, slashes, and underscores are allowed", *g.Branch)
	}
	if g.Remote.isString {
		if g.Remote.name == "" {
			return errors.New("git.remote string cannot be empty; use 'true' to auto-detect")
		}
		if !gitRefRe.MatchString(g.Remote.name) {
			return fmt.Errorf("invalid git.remote name %q: only alphanumeric characters, hyphens, dots, slashes, and underscores are allowed", g.Remote.name)
		}
	}

	if g.Branch == nil && g.Commit == nil {
		return errors.New("git: must specify either 'branch' or 'commit'")
	}
	if g.Branch != nil && g.Commit != nil {
		return errors.New("git: 'branch' and 'commit' are mutually exclusive — specify only one")
	}
	if g.Remote.truthy() && g.Branch == nil {
		return errors.New("git.remote requires git.branch (only valid with branch refs)")
	}
	return nil
}

// gitRemote is git.remote: false (default, use local HEAD), true (auto-detect the
// remote), or a remote name string.
type gitRemote struct {
	set      bool
	isString bool
	name     string
	enabled  bool
}

func (r *gitRemote) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.ScalarNode {
		return errors.New("git.remote must be a boolean or a remote name string")
	}
	r.set = true
	if node.Tag == "!!bool" {
		return node.Decode(&r.enabled)
	}
	r.isString = true
	r.name = node.Value
	return nil
}

// truthy reports whether remote requests a remote fetch (mirrors Python's
// truthiness of the bool|str union).
func (r *gitRemote) truthy() bool {
	if r.isString {
		return r.name != ""
	}
	return r.enabled
}

// permission is a DABs-compatible permission grant: exactly one principal plus a
// level.
type permission struct {
	UserName             *string `yaml:"user_name"`
	GroupName            *string `yaml:"group_name"`
	ServicePrincipalName *string `yaml:"service_principal_name"`
	// Level is a databricks PermissionLevel (e.g. CAN_VIEW, CAN_MANAGE). Enum
	// membership is validated server-side; here we only require it to be set.
	Level string `yaml:"level"`
}

func (p *permission) validate() error {
	principals := map[string]*string{
		"user_name":              p.UserName,
		"group_name":             p.GroupName,
		"service_principal_name": p.ServicePrincipalName,
	}
	var set []string
	for name, val := range principals {
		if val != nil {
			set = append(set, name)
		}
	}
	switch len(set) {
	case 0:
		return errors.New("permissions: one of 'user_name', 'group_name', or 'service_principal_name' must be specified")
	case 1:
		name := set[0]
		if strings.TrimSpace(*principals[name]) == "" {
			return fmt.Errorf("permissions: '%s' cannot be empty", name)
		}
	default:
		return errors.New("permissions: only one of 'user_name', 'group_name', or 'service_principal_name' can be specified")
	}

	if strings.TrimSpace(p.Level) == "" {
		return errors.New("permissions: 'level' is required")
	}
	return nil
}
