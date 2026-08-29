package aircmd

import (
	"errors"
	"fmt"
	"io"
	"maps"
	"reflect"
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

// gitRefRe guards the branch name against command injection (it flows into git
// exec args). Only safe ref characters are allowed.
var gitRefRe = regexp.MustCompile(`^[\w./-]+$`)

// Canonical UUID (8-4-4-4-12 hex). Usage policy ids are server-generated UUIDs,
// so an obviously-wrong value (e.g. a policy name pasted into usage_policy_id)
// is rejected up front with a hint pointing at usage_policy_name.
var uuidRe = regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

// runConfig is the top-level run YAML schema: experiment_name + compute /
// environment / code_source plus the command and run options.
type runConfig struct {
	ExperimentName string             `yaml:"experiment_name" help:"Name of the experiment. Becomes the Jobs API task key: max 100 characters, alphanumerics, hyphens, and underscores only." required:"yes"`
	Compute        *computeConfig     `yaml:"compute" help:"Which accelerators to run on and how many." required:"yes"`
	Environment    *environmentConfig `yaml:"environment" help:"Python dependencies, or a custom Docker image, for the run's runtime."`
	Command        *string            `yaml:"command" help:"Shell command that starts the workload. Max 1000 lines; move longer logic into a script under code_source." required:"yes"`
	EnvVariables   map[string]string  `yaml:"env_variables" help:"Plain environment variables, as NAME: value. A name here cannot also appear in secrets."`
	Secrets        map[string]string  `yaml:"secrets" help:"Environment variables sourced from secrets, as NAME: scope/key."`
	CodeSource     *codeSourceConfig  `yaml:"code_source" help:"Local code to upload and make available to the run."`
	// MaxRetries defaults to 3 when unset; default-filling is a normalization
	// concern handled at launch, so a nil pointer is left as-is here.
	MaxRetries                *int           `yaml:"max_retries" help:"How many times to retry a failed run. Must be >= 0. Defaults to 3 when unset."`
	TimeoutMinutes            *int           `yaml:"timeout_minutes" help:"Wall-clock limit for the run in minutes. Must be >= 1."`
	IdempotencyToken          *string        `yaml:"idempotency_token" help:"Reuse token: a repeat submission with the same token returns the existing run instead of starting another. Max 64 characters."`
	Parameters                map[string]any `yaml:"parameters" help:"Free-form values passed through to the workload. Any nested structure is allowed."`
	MLflowRunName             *string        `yaml:"mlflow_run_name" help:"Name for the MLflow run. Max 100 characters, alphanumerics, hyphens, and underscores only."`
	MLflowExperimentDirectory *string        `yaml:"mlflow_experiment_directory" help:"Workspace directory holding the MLflow experiment. Must start with /Workspace."`
	MLflowArtifactLocation    *string        `yaml:"mlflow_artifact_location" help:"DBFS location where MLflow artifacts are written. A /Volumes path is normalized to dbfs:/Volumes/... ."`
	Permissions               []permission   `yaml:"permissions" help:"Who may view or manage the run, as a list of principal plus level grants."`
	UsagePolicyName           *string        `yaml:"usage_policy_name" help:"Usage policy to bill the run to, by name. Max 127 characters. Mutually exclusive with usage_policy_id."`
	UsagePolicyID             *string        `yaml:"usage_policy_id" help:"Usage policy to bill the run to, by id. Mutually exclusive with usage_policy_name."`
	// Schedule turns the workload into a recurring, persistent job: `air run`
	// creates (or updates) a scheduled job instead of a one-time run (see run.go).
	Schedule *scheduleConfig `yaml:"schedule" help:"Run the workload on a recurring cron schedule as a persistent job, instead of a one-time run."`
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

	if c.MLflowArtifactLocation != nil {
		v := strings.TrimSpace(*c.MLflowArtifactLocation)
		if v == "" {
			return errors.New("mlflow_artifact_location cannot be empty")
		}
		if strings.HasPrefix(v, "/Volumes/") {
			v = "dbfs:" + v
		}
		if !strings.HasPrefix(v, "dbfs:/") {
			return fmt.Errorf("mlflow_artifact_location must be a dbfs: URI, got: %s", v)
		}
		*c.MLflowArtifactLocation = v
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
	if c.UsagePolicyID != nil {
		v := strings.TrimSpace(*c.UsagePolicyID)
		if v == "" {
			return errors.New("usage_policy_id must not be empty")
		}
		if !uuidRe.MatchString(v) {
			return fmt.Errorf("usage_policy_id must be a UUID (for example, '12345678-90ab-cdef-1234-567890abcdef'), got: %s. To assign a policy by name instead, use usage_policy_name", v)
		}
	}

	if c.Schedule != nil {
		if err := c.Schedule.validate(); err != nil {
			return err
		}
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
	Dependencies dependencies       `yaml:"dependencies" help:"Inline list of packages to install. Not allowed alongside docker_image."`
	Version      stringOrInt        `yaml:"version" help:"Client image version to pin. Only valid alongside inline dependencies."`
	DockerImage  *dockerImageConfig `yaml:"docker_image" help:"Custom image supplying the whole runtime. Not allowed alongside dependencies or version."`
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

	// version pins the client image version, which is only meaningful alongside an
	// inline dependency set.
	if e.Version.set && !e.Dependencies.set {
		return errors.New("'environment.version' requires inline 'dependencies' (a list of packages)")
	}
	if e.Version.set {
		version, err := validateRuntimeVersion(e.Version.raw, "environment.version")
		if err != nil {
			return err
		}
		e.Version.raw = version
	}

	return nil
}

// dependencies is environment.dependencies: an inline list of packages. A scalar
// (e.g. a path to a requirements file) is rejected — the list may itself reference
// a requirements.txt, but dependencies must be given as a list.
type dependencies struct {
	set  bool
	list []string
}

func (d *dependencies) UnmarshalYAML(node *yaml.Node) error {
	if node.Kind != yaml.SequenceNode {
		return errors.New("environment.dependencies must be a list of packages or reference a requirements.txt (see https://docs.databricks.com/aws/en/machine-learning/ai-runtime/cli/yaml-config#reference). A direct file reference is not supported")
	}
	d.set = true
	return node.Decode(&d.list)
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
	URL string `yaml:"url" help:"Fully qualified image URL, e.g. myregistry.io/team/train:v3." required:"when environment.docker_image is set"`
	// "latest" re-checks the registry for the tag's newest digest each run;
	// "auto" (default) reuses the existing registration.
	TagPolicy string `yaml:"tag_policy" help:"\"auto\" (default) reuses the existing registration; \"latest\" re-checks the registry for the tag's newest digest each run."`
	// Credentials for a private image that "latest" re-resolves; discovered from
	// the local Docker config when unset.
	CredentialsScope string `yaml:"credentials_scope" help:"Secret scope holding registry credentials for a private image re-resolved under tag_policy \"latest\"; discovered from the local Docker config when unset."`
	CredentialsKey   string `yaml:"credentials_key" help:"Secret key (paired with credentials_scope) for private-image registry credentials."`
}

const (
	dockerTagPolicyAuto   = "auto"
	dockerTagPolicyLatest = "latest"
)

func (d *dockerImageConfig) validate() error {
	// Store the trimmed values: URL rides the submitted task, and the credential
	// pairing check below must not treat blank-but-present as set.
	d.URL = strings.TrimSpace(d.URL)
	d.CredentialsScope = strings.TrimSpace(d.CredentialsScope)
	d.CredentialsKey = strings.TrimSpace(d.CredentialsKey)

	if d.URL == "" {
		return errors.New("docker_image.url cannot be empty")
	}

	switch strings.ToLower(strings.TrimSpace(d.TagPolicy)) {
	case "", dockerTagPolicyAuto, dockerTagPolicyLatest:
	default:
		return fmt.Errorf("invalid docker_image.tag_policy %q: must be %q or %q", d.TagPolicy, dockerTagPolicyAuto, dockerTagPolicyLatest)
	}

	if (d.CredentialsScope != "") != (d.CredentialsKey != "") {
		return errors.New("docker_image.credentials_scope and docker_image.credentials_key must be provided together")
	}

	// Credentials are only consulted when re-resolving the tag, so accepting them
	// under the default policy would silently ignore them.
	if d.CredentialsScope != "" && !d.wantsLatest() {
		return fmt.Errorf("docker_image.credentials_scope/credentials_key only apply with tag_policy %q; the image is otherwise used as already registered", dockerTagPolicyLatest)
	}
	return nil
}

// wantsLatest reports whether the image should be re-resolved before the run.
func (d *dockerImageConfig) wantsLatest() bool {
	return strings.EqualFold(strings.TrimSpace(d.TagPolicy), dockerTagPolicyLatest)
}

// codeSourceConfig is the `code_source` block. Only the "snapshot" type exists.
type codeSourceConfig struct {
	Type     string                `yaml:"type" help:"Kind of code source. Must be \"snapshot\", the only supported type." required:"when code_source is set"`
	Snapshot *snapshotSourceConfig `yaml:"snapshot" help:"Which local directory to archive and upload." required:"when code_source.type is \"snapshot\""`
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
	RootPath     string   `yaml:"root_path" help:"Local directory to archive, relative or absolute." required:"when code_source.snapshot is set"`
	RemoteVolume *string  `yaml:"remote_volume" help:"Volume to upload the archive to. Must start with /Volumes/."`
	Git          *gitRef  `yaml:"git" help:"Pin the snapshot to a specific git revision."`
	IncludePaths []string `yaml:"include_paths" help:"Restrict the archive to these paths, relative to root_path and without \"..\". Omit to include everything."`
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

// scheduleConfig mirrors the Jobs CronSchedule proto so it maps 1:1 onto the
// bundle job's schedule block (see convert_to_dabs.go). pause_status is optional
// and defaults to UNPAUSED, matching the Jobs default.
type scheduleConfig struct {
	QuartzCronExpression string `yaml:"quartz_cron_expression" help:"Quartz cron expression for the schedule, e.g. '0 0 9 * * ?' (daily at 9am)." required:"yes"`
	TimezoneID           string `yaml:"timezone_id" help:"Timezone the cron expression is evaluated in, e.g. 'America/Los_Angeles' or 'UTC'." required:"yes"`
	PauseStatus          string `yaml:"pause_status" help:"Whether the schedule starts PAUSED or UNPAUSED. Optional; defaults to UNPAUSED."`
}

func (s *scheduleConfig) validate() error {
	if strings.TrimSpace(s.QuartzCronExpression) == "" {
		return errors.New("schedule.quartz_cron_expression is required")
	}
	if strings.TrimSpace(s.TimezoneID) == "" {
		return errors.New("schedule.timezone_id is required (for example, 'America/Los_Angeles' or 'UTC')")
	}
	switch s.PauseStatus {
	case "", "PAUSED", "UNPAUSED":
	default:
		return fmt.Errorf("schedule.pause_status must be PAUSED or UNPAUSED, got %q", s.PauseStatus)
	}
	return nil
}

// gitRef pins a snapshot to a specific git ref. branch and commit are mutually
// exclusive; remote is only meaningful with branch.
type gitRef struct {
	Branch *string   `yaml:"branch" help:"Branch to pin to, resolved to its local HEAD. Mutually exclusive with commit." required:"one of branch or commit"`
	Commit *string   `yaml:"commit" help:"Commit to pin to. Mutually exclusive with branch." required:"one of branch or commit"`
	Remote gitRemote `yaml:"remote" help:"No longer supported: the snapshot archives your local copy. Only false is accepted; use commit to pin a revision."`
}

func (g *gitRef) validate() error {
	if g.Branch != nil && !gitRefRe.MatchString(*g.Branch) {
		return fmt.Errorf("invalid git.branch format %q: only alphanumeric characters, hyphens, dots, slashes, and underscores are allowed", *g.Branch)
	}

	// The remote-fetch path (fetching a branch's remote HEAD) is deprecated: the
	// snapshot archives the local copy only. A truthy git.remote (a name or `true`)
	// is rejected; `remote: false` is the default (local HEAD) and stays valid.
	if g.Remote.truthy() {
		return errors.New("git.remote is no longer supported: the snapshot archives your local copy, so a branch resolves to its local HEAD. To deploy a specific committed revision, use git.commit")
	}

	if g.Branch == nil && g.Commit == nil {
		return errors.New("git: must specify either 'branch' or 'commit'")
	}
	if g.Branch != nil && g.Commit != nil {
		return errors.New("git: 'branch' and 'commit' are mutually exclusive — specify only one")
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
	UserName             *string `yaml:"user_name" help:"Grant to this user, by email. Exactly one principal field per grant." required:"one principal per grant"`
	GroupName            *string `yaml:"group_name" help:"Grant to this group, by name. Exactly one principal field per grant." required:"one principal per grant"`
	ServicePrincipalName *string `yaml:"service_principal_name" help:"Grant to this service principal, by name. Exactly one principal field per grant." required:"one principal per grant"`
	// Level is a databricks PermissionLevel (e.g. CAN_VIEW, CAN_MANAGE). Enum
	// membership is validated server-side; here we only require it to be set.
	Level string `yaml:"level" help:"Permission level to grant, e.g. CAN_VIEW or CAN_MANAGE. Validated server-side." required:"when a grant is listed"`
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

// Below: `air run -h config.<field>`, which documents the schema above from its
// yaml/help/required struct tags.

// configHelpRoot is the optional leading segment of a help path (`config.compute`
// or bare `compute`).
const configHelpRoot = "config"

// writeConfigFieldHelp resolves a dotted config path and writes its docs.
func writeConfigFieldHelp(w io.Writer, path string) error {
	field, err := resolveConfigField(path)
	if err != nil {
		return err
	}
	renderConfigField(w, field)
	return nil
}

// freeFormConfigFields hold free-form maps, so path resolution stops at them:
// their keys are chosen by the user, not the schema.
var freeFormConfigFields = map[string]bool{
	"parameters":    true,
	"env_variables": true,
	"secrets":       true,
}

// configTypeNames labels the polymorphic unions, whose YAML shape reflection
// can't see (unexported fields filled by a custom UnmarshalYAML).
var configTypeNames = map[reflect.Type]string{
	reflect.TypeFor[dependencies](): "list of strings",
	reflect.TypeFor[stringOrInt]():  "string or int",
	reflect.TypeFor[gitRemote]():    "bool or string",
}

// configField is one resolved node of the run config schema.
type configField struct {
	path     string
	typeName string
	required string
	help     string
	// freeForm marks a user-keyed map (parameters/secrets/env_variables): no
	// children, yet any sub-path into it is valid.
	freeForm bool
	children []configField // nil for a leaf
}

// configSchema is the single reflection walk over runConfig; both
// `-h config.<field>` and --override path validation resolve against it.
func configSchema() configField {
	return configField{
		path:     configHelpRoot,
		help:     "The run YAML schema. Pass a field path for details, e.g. " + configHelpRoot + ".compute.accelerator_type.",
		children: describeStruct(reflect.TypeFor[runConfig](), configHelpRoot),
	}
}

// resolveConfigField resolves a dotted YAML path against the run config schema.
// The leading "config." is optional; an empty path describes the whole schema.
func resolveConfigField(path string) (configField, error) {
	trimmed := strings.TrimPrefix(strings.TrimPrefix(path, configHelpRoot), ".")
	root := configSchema()
	if trimmed == "" {
		return root, nil
	}

	current := root
	for i, part := range strings.Split(trimmed, ".") {
		if current.freeForm {
			// Keys are user-defined, so the map is the most specific node.
			return configField{}, fmt.Errorf("%q holds user-defined keys, so %q is not part of the schema; see %q instead", current.path, part, current.path)
		}
		if len(current.children) == 0 {
			return configField{}, fmt.Errorf("%q is not an object, so it has no field %q", current.path, part)
		}
		child, ok := findConfigChild(current.children, part)
		if !ok {
			return configField{}, unknownConfigFieldError(current, part, strings.Split(trimmed, ".")[:i+1])
		}
		current = child
	}
	return current, nil
}

// findConfigChild looks up an immediate child by its YAML name.
func findConfigChild(children []configField, name string) (configField, bool) {
	for _, c := range children {
		if configLeafName(c.path) == name {
			return c, true
		}
	}
	return configField{}, false
}

// configLeafName returns the last segment of a dotted path.
func configLeafName(path string) string {
	_, leaf, found := cutLast(path, ".")
	if !found {
		return path
	}
	return leaf
}

// cutLast splits s around the final instance of sep.
func cutLast(s, sep string) (before, after string, found bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

// unknownConfigFieldError reports an unresolvable segment, naming the valid
// siblings and, when one is close enough, a suggestion.
func unknownConfigFieldError(parent configField, part string, matched []string) error {
	names := make([]string, 0, len(parent.children))
	for _, c := range parent.children {
		names = append(names, configLeafName(c.path))
	}
	slices.Sort(names)

	msg := fmt.Sprintf("unknown config field %q", configHelpRoot+"."+strings.Join(matched, "."))
	if suggestion, ok := closestConfigField(part, names); ok {
		msg += fmt.Sprintf("; did you mean %q?", suggestion)
	}
	return fmt.Errorf("%s\n\nfields under %q are: %s", msg, parent.path, strings.Join(names, ", "))
}

// closestConfigField returns the nearest candidate by edit distance, if one is
// close enough to be worth suggesting.
func closestConfigField(name string, candidates []string) (string, bool) {
	best, bestDist := "", 0
	for _, c := range candidates {
		d := configEditDistance(name, c)
		// Allow roughly a third of the name to differ, and always accept a
		// single edit so short names still get a suggestion.
		limit := max(len(c)/3, 1)
		if d <= limit && (best == "" || d < bestDist) {
			best, bestDist = c, d
		}
	}
	return best, best != ""
}

// configEditDistance computes the Levenshtein distance between two strings.
func configEditDistance(a, b string) int {
	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min(min(curr[j-1]+1, prev[j]+1), prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

// describeStruct reads a struct's yaml/help/required tags into configFields,
// recursing into nested objects. Declaration order matches validate()'s errors.
func describeStruct(t reflect.Type, prefix string) []configField {
	var out []configField
	for f := range t.Fields() {
		tag := f.Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name == "" || name == "-" {
			continue
		}

		field := configField{
			path:     prefix + "." + name,
			typeName: configTypeName(f.Type),
			required: f.Tag.Get("required"),
			help:     f.Tag.Get("help"),
			freeForm: freeFormConfigFields[name],
		}
		if nested := underlyingConfigStruct(f.Type); nested != nil && !field.freeForm {
			field.children = describeStruct(nested, field.path)
		}
		out = append(out, field)
	}
	return out
}

// configTypeName renders a field's YAML-facing type.
func configTypeName(t reflect.Type) string {
	if name, ok := configTypeNames[t]; ok {
		return name
	}
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string"
	case reflect.Int:
		return "int"
	case reflect.Bool:
		return "bool"
	case reflect.Slice:
		return "list of " + configTypeName(t.Elem())
	case reflect.Map:
		return fmt.Sprintf("map of %s to %s", configTypeName(t.Key()), configTypeName(t.Elem()))
	case reflect.Struct:
		return "object"
	case reflect.Interface:
		return "any"
	default:
		return t.Kind().String()
	}
}

// underlyingConfigStruct unwraps pointer/slice indirection and returns the struct
// type a field decodes into, or nil if it is not a struct. The polymorphic unions
// are excluded: they are structs, but their YAML shape is scalar or list.
func underlyingConfigStruct(t reflect.Type) reflect.Type {
	if _, ok := configTypeNames[t]; ok {
		return nil
	}
	for t.Kind() == reflect.Pointer || t.Kind() == reflect.Slice {
		t = t.Elem()
	}
	if t.Kind() == reflect.Struct {
		return t
	}
	return nil
}

// renderConfigField writes a resolved field's documentation. An object lists its
// immediate children; a leaf gets its type, required-ness, and description.
func renderConfigField(w io.Writer, f configField) {
	fmt.Fprintf(w, "%s\n", f.path)
	if f.help != "" {
		fmt.Fprintf(w, "  %s\n", f.help)
	}

	if len(f.children) == 0 {
		fmt.Fprintf(w, "\n  Type:     %s\n", f.typeName)
		required := f.required
		if required == "" {
			required = "no"
		}
		fmt.Fprintf(w, "  Required: %s\n", required)
		return
	}

	width := 0
	for _, c := range f.children {
		width = max(width, len(configLeafName(c.path)))
	}
	fmt.Fprintf(w, "\n  Fields:\n")
	for _, c := range f.children {
		fmt.Fprintf(w, "    %-*s  %s\n", width, configLeafName(c.path), configFieldSummary(c))
	}
	fmt.Fprintf(w, "\nUse \"-h %s.<field>\" for details on a field.\n", f.path)
}

// configFieldSummary is the one-line description used in a field listing: the
// first sentence of the help text, annotated when the field is required.
func configFieldSummary(f configField) string {
	summary := firstSentence(f.help)
	if f.required == "yes" {
		summary = "(required) " + summary
	}
	return summary
}

// sentenceAbbreviations end in a period that does not close a sentence, so
// firstSentence must not break on them.
var sentenceAbbreviations = []string{"e.g.", "i.e.", "etc.", "vs.", "cf."}

// firstSentence returns s up to and including the first sentence-ending period,
// i.e. the first ". " boundary not immediately preceded by a known abbreviation.
// Returns s unchanged when it holds a single sentence.
func firstSentence(s string) string {
	for i := 0; i+1 < len(s); i++ {
		if s[i] != '.' || s[i+1] != ' ' {
			continue
		}
		candidate := s[:i+1]
		if slices.ContainsFunc(sentenceAbbreviations, func(a string) bool {
			return strings.HasSuffix(candidate, a)
		}) {
			continue
		}
		return candidate
	}
	return s
}
