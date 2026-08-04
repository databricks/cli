package aircmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig writes content to a temp YAML file and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// minimalConfig is the smallest valid config: the three required pieces.
const minimalConfig = `
experiment_name: my-run
command: python train.py
compute:
  accelerator_type: GPU_1xH100
  num_accelerators: 1
`

func TestLoadRunConfig_Minimal(t *testing.T) {
	cfg, err := loadRunConfig(writeConfig(t, minimalConfig))
	require.NoError(t, err)
	assert.Equal(t, "my-run", cfg.ExperimentName)
	require.NotNil(t, cfg.Command)
	assert.Equal(t, "python train.py", *cfg.Command)
	require.NotNil(t, cfg.Compute)
	assert.Equal(t, "GPU_1xH100", cfg.Compute.AcceleratorType)
	assert.Equal(t, 1, cfg.Compute.NumAccelerators)
}

func TestLoadRunConfig_FullFeatured(t *testing.T) {
	cfg, err := loadRunConfig(writeConfig(t, `
experiment_name: full_run
command: |
  python train.py
  echo done
compute:
  accelerator_type: GPU_8xH100
  num_accelerators: 16
environment:
  dependencies:
    - torch==2.3.0
    - numpy
  version: 5
env_variables:
  FOO: bar
secrets:
  HF_TOKEN: my_scope/hf_token
code_source:
  type: snapshot
  snapshot:
    root_path: project_root/src
    remote_volume: /Volumes/main/default/code
    git:
      branch: main
    include_paths:
      - src
      - configs/train.yaml
max_retries: 5
timeout_minutes: 120
idempotency_token: abc-123
mlflow_run_name: full_run_v2
mlflow_experiment_directory: /Workspace/Users/me/exp
usage_policy_name: my-policy
permissions:
  - group_name: users
    level: CAN_VIEW
  - user_name: alice@example.com
    level: CAN_MANAGE
`))
	require.NoError(t, err)
	assert.Equal(t, gpuType8xH100, gpuType(cfg.Compute.AcceleratorType))
	require.NotNil(t, cfg.Environment)
	assert.True(t, cfg.Environment.Dependencies.isList)
	assert.Equal(t, []string{"torch==2.3.0", "numpy"}, cfg.Environment.Dependencies.list)
	assert.True(t, cfg.Environment.Version.set)
	assert.Equal(t, "5", cfg.Environment.Version.raw)
	require.NotNil(t, cfg.CodeSource)
	require.NotNil(t, cfg.CodeSource.Snapshot)
	require.NotNil(t, cfg.CodeSource.Snapshot.Git)
	require.NotNil(t, cfg.CodeSource.Snapshot.Git.Branch)
	assert.Equal(t, "main", *cfg.CodeSource.Snapshot.Git.Branch)
	assert.Len(t, cfg.Permissions, 2)
}

// TestLoadRunConfig_PolymorphicFields exercises the str|list, str|int, and
// bool|str unions decoded by custom UnmarshalYAML.
func TestLoadRunConfig_PolymorphicFields(t *testing.T) {
	t.Run("dependencies as string path", func(t *testing.T) {
		cfg, err := loadRunConfig(writeConfig(t, minimalConfig+`
environment:
  dependencies: requirements.yaml
`))
		require.NoError(t, err)
		assert.True(t, cfg.Environment.Dependencies.set)
		assert.False(t, cfg.Environment.Dependencies.isList)
		assert.Equal(t, "requirements.yaml", cfg.Environment.Dependencies.path)
	})

	t.Run("git remote as bool true is rejected", func(t *testing.T) {
		_, err := loadRunConfig(writeConfig(t, minimalConfig+`
code_source:
  type: snapshot
  snapshot:
    root_path: .
    git:
      branch: main
      remote: true
`))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git.remote is no longer supported")
	})

	t.Run("git remote defaults to false when unset", func(t *testing.T) {
		cfg, err := loadRunConfig(writeConfig(t, minimalConfig+`
code_source:
  type: snapshot
  snapshot:
    root_path: .
    git:
      commit: deadbeef
`))
		require.NoError(t, err)
		assert.False(t, cfg.CodeSource.Snapshot.Git.Remote.truthy())
	})
}

func TestLoadRunConfig_UnknownFieldRejected(t *testing.T) {
	tests := []struct {
		name    string
		extra   string
		errFrag string
	}{
		{"top-level typo", "extra_field: nope\n", "extra_field"},
		// priority was intentionally dropped from the schema (pool-only concept).
		{"dropped priority field", "priority: 100\n", "priority"},
		// _bases_ composition is not yet ported, so it surfaces as unknown.
		{"unported _bases_", "_bases_: [base.yaml]\n", "_bases_"},
		{"nested typo", "environment:\n  bogus: 1\n", "bogus"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRunConfig(writeConfig(t, minimalConfig+tt.extra))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestLoadRunConfig_Errors(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		errFrag string
	}{
		{
			"missing experiment_name",
			"command: x\ncompute:\n  accelerator_type: GPU_1xH100\n  num_accelerators: 1\n",
			"experiment_name cannot be empty",
		},
		{
			"experiment_name bad chars",
			"experiment_name: my.run\ncommand: x\ncompute:\n  accelerator_type: GPU_1xH100\n  num_accelerators: 1\n",
			"invalid experiment_name",
		},
		{
			"missing compute",
			"experiment_name: r\ncommand: x\n",
			"compute: section is required",
		},
		{
			"missing command",
			"experiment_name: r\ncompute:\n  accelerator_type: GPU_1xH100\n  num_accelerators: 1\n",
			"command is required",
		},
		{
			"bad gpu type",
			"experiment_name: r\ncommand: x\ncompute:\n  accelerator_type: a100\n  num_accelerators: 1\n",
			"invalid GPU type",
		},
		{
			"num_accelerators not a multiple",
			"experiment_name: r\ncommand: x\ncompute:\n  accelerator_type: GPU_8xH100\n  num_accelerators: 3\n",
			"must be a multiple of 8",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := loadRunConfig(writeConfig(t, tt.yaml))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

// TestRunConfigValidate_FieldRules unit-tests validation rules directly, away
// from YAML decoding, to keep each rule's failure mode explicit.
func TestRunConfigValidate_FieldRules(t *testing.T) {
	str := func(s string) *string { return &s }
	intp := func(i int) *int { return &i }
	base := func() *runConfig {
		return &runConfig{
			ExperimentName: "r",
			Command:        str("x"),
			Compute:        &computeConfig{AcceleratorType: "GPU_1xH100", NumAccelerators: 1},
		}
	}

	tests := []struct {
		name    string
		mutate  func(c *runConfig)
		errFrag string
	}{
		{"ok baseline", func(c *runConfig) {}, ""},
		{"empty command", func(c *runConfig) { c.Command = str("   ") }, "command cannot be empty"},
		{"negative max_retries", func(c *runConfig) { c.MaxRetries = intp(-1) }, "max_retries must be >= 0"},
		{"zero timeout", func(c *runConfig) { c.TimeoutMinutes = intp(0) }, "timeout_minutes must be >= 1"},
		{"empty idempotency", func(c *runConfig) { c.IdempotencyToken = str("  ") }, "idempotency_token cannot be empty"},
		{"long idempotency", func(c *runConfig) { c.IdempotencyToken = str(string(make([]byte, 65))) }, "64 characters or less"},
		{"bad mlflow_run_name", func(c *runConfig) { c.MLflowRunName = str("bad name") }, "invalid mlflow_run_name"},
		{"bad experiment dir", func(c *runConfig) { c.MLflowExperimentDirectory = str("/Users/me") }, "must start with '/Workspace'"},
		{"empty usage policy", func(c *runConfig) { c.UsagePolicyName = str(" ") }, "usage_policy_name must not be empty"},
		{"bad secret ref", func(c *runConfig) { c.Secrets = map[string]string{"T": "noslash"} }, "expected format 'scope/key'"},
		{"empty secret scope", func(c *runConfig) { c.Secrets = map[string]string{"T": "/key"} }, "scope and key cannot be empty"},
		{"env var and secret collide", func(c *runConfig) {
			c.EnvVariables = map[string]string{"TOK": "v"}
			c.Secrets = map[string]string{"TOK": "scope/key"}
		}, `"TOK" is set in both env_variables and secrets`},
		{"long mlflow_run_name", func(c *runConfig) { c.MLflowRunName = str(strings.Repeat("a", 101)) }, "100 characters or less"},
		{"usage policy name and id", func(c *runConfig) {
			c.UsagePolicyName = str("p")
			c.UsagePolicyID = str("id")
		}, "mutually exclusive"},
		{"empty usage_policy_id", func(c *runConfig) { c.UsagePolicyID = str("  ") }, "usage_policy_id must not be empty"},
		{"non-uuid usage_policy_id", func(c *runConfig) { c.UsagePolicyID = str("policy-uuid") }, "usage_policy_id must be a UUID"},
		// A name pasted into the id field gets pointed at the right field.
		{"policy name in usage_policy_id", func(c *runConfig) { c.UsagePolicyID = str("team-a") }, "use usage_policy_name"},
		{"uuid usage_policy_id alone is ok", func(c *runConfig) { c.UsagePolicyID = str("12345678-90ab-cdef-1234-567890abcdef") }, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := base()
			tt.mutate(c)
			err := c.validate()
			if tt.errFrag == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestEnvironmentConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		env     environmentConfig
		errFrag string
	}{
		{
			"docker image alone ok",
			environmentConfig{DockerImage: &dockerImageConfig{URL: "org/repo:tag"}},
			"",
		},
		{
			"docker image with deps conflicts",
			environmentConfig{
				DockerImage:  &dockerImageConfig{URL: "org/repo:tag"},
				Dependencies: dependencies{set: true, isList: true, list: []string{"torch"}},
			},
			"not allowed: dependencies",
		},
		{
			"empty docker url",
			environmentConfig{DockerImage: &dockerImageConfig{URL: "  "}},
			"docker_image.url cannot be empty",
		},
		{
			"version with file deps",
			environmentConfig{
				Version:      stringOrInt{set: true, raw: "5"},
				Dependencies: dependencies{set: true, isList: false, path: "req.yaml"},
			},
			"only valid with inline dependencies",
		},
		{
			"version without deps",
			environmentConfig{Version: stringOrInt{set: true, raw: "5"}},
			"requires inline 'dependencies'",
		},
		{
			"version with inline deps ok",
			environmentConfig{
				Version:      stringOrInt{set: true, raw: "5"},
				Dependencies: dependencies{set: true, isList: true, list: []string{"torch"}},
			},
			"",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.env.validate()
			if tt.errFrag == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestGitRefValidate(t *testing.T) {
	str := func(s string) *string { return &s }
	tests := []struct {
		name    string
		ref     gitRef
		errFrag string
	}{
		{"branch only ok", gitRef{Branch: str("main")}, ""},
		{"commit only ok", gitRef{Commit: str("abc123")}, ""},
		{"remote false is ok", gitRef{Branch: str("main"), Remote: gitRemote{set: true, enabled: false}}, ""},
		{"neither branch nor commit", gitRef{}, "must specify either 'branch' or 'commit'"},
		{"both branch and commit", gitRef{Branch: str("main"), Commit: str("abc")}, "mutually exclusive"},
		{"remote true rejected", gitRef{Branch: str("main"), Remote: gitRemote{set: true, enabled: true}}, "git.remote is no longer supported"},
		{"remote name rejected", gitRef{Branch: str("main"), Remote: gitRemote{set: true, isString: true, name: "origin"}}, "git.remote is no longer supported"},
		{"bad branch chars", gitRef{Branch: str("bad branch")}, "invalid git.branch"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ref.validate()
			if tt.errFrag == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestSnapshotSourceConfigValidate(t *testing.T) {
	tests := []struct {
		name    string
		snap    snapshotSourceConfig
		errFrag string
	}{
		{"ok", snapshotSourceConfig{RootPath: "src"}, ""},
		{"empty root_path", snapshotSourceConfig{RootPath: "  "}, "root_path cannot be empty"},
		{"bad volume", snapshotSourceConfig{RootPath: "src", RemoteVolume: new("/mnt/x")}, "must start with '/Volumes/'"},
		{"empty include list", snapshotSourceConfig{RootPath: "src", IncludePaths: []string{}}, "cannot be an empty list"},
		{"absolute include", snapshotSourceConfig{RootPath: "src", IncludePaths: []string{"/etc"}}, "must be relative"},
		{"traversal include", snapshotSourceConfig{RootPath: "src", IncludePaths: []string{"../x"}}, "'..' traversal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.snap.validate()
			if tt.errFrag == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestPermissionValidate(t *testing.T) {
	str := func(s string) *string { return &s }
	tests := []struct {
		name    string
		perm    permission
		errFrag string
	}{
		{"ok user", permission{UserName: str("alice@example.com"), Level: "CAN_VIEW"}, ""},
		{"no principal", permission{Level: "CAN_VIEW"}, "must be specified"},
		{"two principals", permission{UserName: str("a"), GroupName: str("g"), Level: "CAN_VIEW"}, "only one of"},
		{"empty principal", permission{UserName: str("  "), Level: "CAN_VIEW"}, "cannot be empty"},
		{"missing level", permission{GroupName: str("users")}, "'level' is required"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.perm.validate()
			if tt.errFrag == "" {
				assert.NoError(t, err)
				return
			}
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.errFrag)
		})
	}
}

func TestLoadRunConfig_FileErrors(t *testing.T) {
	t.Run("missing file", func(t *testing.T) {
		_, err := loadRunConfig(filepath.Join(t.TempDir(), "nope.yaml"))
		assert.Error(t, err)
	})
	t.Run("empty file", func(t *testing.T) {
		_, err := loadRunConfig(writeConfig(t, ""))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "is empty")
	})
}

func TestResolveConfigField(t *testing.T) {
	tests := []struct {
		name         string
		path         string
		wantPath     string
		wantType     string
		wantRequired string
	}{
		{"leaf", "config.compute.accelerator_type", "config.compute.accelerator_type", "string", ""},
		{"bare path", "compute.accelerator_type", "config.compute.accelerator_type", "string", ""},
		{"top-level required", "config.experiment_name", "config.experiment_name", "string", "yes"},
		{"int leaf", "config.max_retries", "config.max_retries", "int", ""},
		{"conditionally required", "config.environment.docker_image.url", "config.environment.docker_image.url", "string", "when environment.docker_image is set"},
		{"through a slice", "config.permissions.level", "config.permissions.level", "string", "when a grant is listed"},
		{"free-form map", "config.parameters", "config.parameters", "map of string to any", ""},
		{"deeply nested", "config.code_source.snapshot.root_path", "config.code_source.snapshot.root_path", "string", "when code_source.snapshot is set"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, err := resolveConfigField(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, field.path)
			assert.Equal(t, tt.wantType, field.typeName)
			assert.Equal(t, tt.wantRequired, field.required)
			assert.NotEmpty(t, field.help)
			assert.Empty(t, field.children)
		})
	}
}

// The union types are structs of unexported fields, so their type label comes
// from configTypeNames rather than reflection, and they must not be walked into.
func TestResolveConfigField_PolymorphicTypes(t *testing.T) {
	tests := []struct{ path, wantType string }{
		{"config.environment.dependencies", "list of strings"},
		{"config.environment.version", "string or int"},
		{"config.code_source.snapshot.git.remote", "bool or string"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			field, err := resolveConfigField(tt.path)
			require.NoError(t, err)
			assert.Equal(t, tt.wantType, field.typeName)
			// A union must not be walked into as an object: its fields are unexported.
			assert.Empty(t, field.children)
		})
	}
}

func TestResolveConfigField_Containers(t *testing.T) {
	root, err := resolveConfigField("config")
	require.NoError(t, err)
	assert.Equal(t, "config", root.path)
	// Children are listed in declaration order, matching the schema.
	assert.Equal(t, "experiment_name", configLeafName(root.children[0].path))
	assert.Equal(t, "compute", configLeafName(root.children[1].path))

	// An empty path describes the whole schema, so `-h config` and a bare
	// prefix agree.
	bare, err := resolveConfigField("")
	require.NoError(t, err)
	assert.Equal(t, root.children, bare.children)

	compute, err := resolveConfigField("config.compute")
	require.NoError(t, err)
	assert.Equal(t, "object", compute.typeName)
	require.Len(t, compute.children, 2)
}

func TestResolveConfigField_Errors(t *testing.T) {
	tests := []struct {
		name      string
		path      string
		wantParts []string
	}{
		{
			name:      "typo suggests the near match",
			path:      "config.compute.acclerator_type",
			wantParts: []string{`unknown config field "config.compute.acclerator_type"`, `did you mean "accelerator_type"?`, "accelerator_type, num_accelerators"},
		},
		{
			name:      "unknown top-level field lists siblings",
			path:      "config.bogus",
			wantParts: []string{`unknown config field "config.bogus"`, "experiment_name"},
		},
		{
			name:      "no suggestion when nothing is close",
			path:      "config.zzzzzzzzzzzz",
			wantParts: []string{`unknown config field "config.zzzzzzzzzzzz"`},
		},
		{
			name:      "free-form keys are not schema fields",
			path:      "config.parameters.learning_rate",
			wantParts: []string{`"config.parameters" holds user-defined keys`, `"learning_rate" is not part of the schema`},
		},
		{
			name:      "scalar has no sub-fields",
			path:      "config.command.foo",
			wantParts: []string{`"config.command" is not an object`, `no field "foo"`},
		},
		{
			name:      "nested typo reports the resolved prefix",
			path:      "config.code_source.snapshot.rootpath",
			wantParts: []string{`unknown config field "config.code_source.snapshot.rootpath"`, `did you mean "root_path"?`},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := resolveConfigField(tt.path)
			require.Error(t, err)
			for _, part := range tt.wantParts {
				assert.Contains(t, err.Error(), part)
			}
		})
	}

	// A distant name must not attract a suggestion.
	_, err := resolveConfigField("config.zzzzzzzzzzzz")
	require.Error(t, err)
	assert.NotContains(t, err.Error(), "did you mean")
}

func TestWriteConfigFieldHelp(t *testing.T) {
	var leaf strings.Builder
	require.NoError(t, writeConfigFieldHelp(&leaf, "config.compute.num_accelerators"))
	assert.Contains(t, leaf.String(), "config.compute.num_accelerators")
	assert.Contains(t, leaf.String(), "Type:     int")
	assert.Contains(t, leaf.String(), "Required: no")

	var container strings.Builder
	require.NoError(t, writeConfigFieldHelp(&container, "config.compute"))
	assert.Contains(t, container.String(), "Fields:")
	assert.Contains(t, container.String(), "num_accelerators")
	// A container lists its fields rather than printing a type/required pair.
	assert.NotContains(t, container.String(), "Required:")
	assert.Contains(t, container.String(), `Use "-h config.compute.<field>" for details`)

	// Required top-level fields are flagged in a listing.
	var root strings.Builder
	require.NoError(t, writeConfigFieldHelp(&root, "config"))
	assert.Contains(t, root.String(), "(required) Which accelerators to run on")

	require.Error(t, writeConfigFieldHelp(&strings.Builder{}, "config.nope"))
}

// Every schema field must document itself, so a new field cannot be added
// without help text. This is the same guard the bundle schema gets from its
// PLACEHOLDER convention in annotations.yml.
func TestConfigFieldsAllDocumented(t *testing.T) {
	root, err := resolveConfigField("config")
	require.NoError(t, err)

	var walk func(fields []configField)
	walk = func(fields []configField) {
		for _, f := range fields {
			assert.NotEmpty(t, f.help, "%s is missing a help: struct tag", f.path)
			assert.NotEmpty(t, f.typeName, "%s has no type name", f.path)
			walk(f.children)
		}
	}
	walk(root.children)
}

func TestFirstSentence(t *testing.T) {
	tests := []struct{ in, want string }{
		{"One sentence with no period", "One sentence with no period"},
		{"First. Second.", "First."},
		{"Which accelerator, e.g. GPU_1xA10. See the docs.", "Which accelerator, e.g. GPU_1xA10."},
		{"Use numpy, torch, etc. Then run.", "Use numpy, torch, etc. Then run."},
		{"Prod i.e. production. Details.", "Prod i.e. production."},
		{"Ends with abbreviation etc.", "Ends with abbreviation etc."},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, firstSentence(tt.in))
	}
}

// Plain -h must fall back to command help. A detached command has no parent to
// inherit a help function from, so this also covers that nil case.
func TestRunCommandHelp_NoConfigPath(t *testing.T) {
	var out strings.Builder
	cmd := newRunCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "Usage:")
	assert.NotContains(t, out.String(), "Fields:")
}

// -h with a config path documents the field instead of the command, and does so
// without the otherwise-required --file.
func TestRunCommandHelp_ConfigPath(t *testing.T) {
	var out strings.Builder
	cmd := newRunCommand()
	cmd.SetOut(&out)
	cmd.SetArgs([]string{"-h", "config.compute.accelerator_type"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, out.String(), "config.compute.accelerator_type")
	assert.Contains(t, out.String(), "Type:     string")
	assert.NotContains(t, out.String(), "Usage:")
}

// An unresolvable path reports to stderr and still exits cleanly, matching how
// cobra treats the help path.
func TestRunCommandHelp_UnknownConfigPath(t *testing.T) {
	var out, errOut strings.Builder
	cmd := newRunCommand()
	cmd.SetOut(&out)
	cmd.SetErr(&errOut)
	cmd.SetArgs([]string{"-h", "config.nope"})
	require.NoError(t, cmd.Execute())
	assert.Contains(t, errOut.String(), `unknown config field "config.nope"`)
	assert.Empty(t, out.String())
}
