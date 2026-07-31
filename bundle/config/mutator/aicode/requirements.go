package aicode

import (
	"bytes"
	"context"
	"fmt"
	"path"

	"github.com/databricks/cli/bundle"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/filer"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/compute"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"go.yaml.in/yaml/v3"
)

// requirementsFileName is the file the AI Runtime entry script reads from the
// command_path directory to install the workload's pip dependencies. The server
// derives its path from command_path, so it must sit next to command.sh.
const requirementsFileName = "requirements.yaml"

// requirementsSpec is the requirements.yaml shape the AI Runtime consumes: a
// client image version and a list of pip requirement lines. It mirrors what the
// Python air CLI synthesizes.
type requirementsSpec struct {
	Version      string   `yaml:"version,omitempty"`
	Dependencies []string `yaml:"dependencies,omitempty"`
}

// SynthesizeRequirements uploads a requirements.yaml next to each AI Runtime task's
// command_path, derived from the job-level serverless environment referenced by the
// task's environment_key. The AI Runtime entry script reads this file (resolved
// from command_path's directory) to set up the workload environment; without it the
// run fails during setup. It runs at deploy time, after command_path has been
// translated to its absolute workspace path.
func SynthesizeRequirements() bundle.Mutator {
	return &synthesizeRequirements{}
}

type synthesizeRequirements struct {
	// client is the filer used for uploads, keyed by the command_path directory.
	// When nil (normal case) a workspace filer is built per directory; only set in
	// tests to inject a recording filer.
	client filer.Filer
}

func (m *synthesizeRequirements) Name() string {
	return "aicode.SynthesizeRequirements"
}

func (m *synthesizeRequirements) Apply(ctx context.Context, b *bundle.Bundle) diag.Diagnostics {
	var diags diag.Diagnostics

	for name, job := range b.Config.Resources.Jobs {
		envs := environmentsByKey(job.Environments)
		for i := range job.Tasks {
			task := &job.Tasks[i]
			if task.AiRuntimeTask == nil {
				continue
			}
			if err := m.synthesizeForTask(ctx, b, name, task, envs); err != nil {
				diags = diags.Extend(diag.FromErr(err))
			}
		}
	}

	return diags
}

// synthesizeForTask uploads requirements.yaml next to task's command_path. It is a
// no-op when the task has no deployment command_path or no matching environment
// spec (the run can still supply deps another way, so this is not an error).
func (m *synthesizeRequirements) synthesizeForTask(ctx context.Context, b *bundle.Bundle, jobName string, task *jobs.Task, envs map[string]*compute.Environment) error {
	if len(task.AiRuntimeTask.Deployments) == 0 {
		return nil
	}
	commandPath := task.AiRuntimeTask.Deployments[0].CommandPath
	if commandPath == "" {
		return nil
	}

	env := envs[task.EnvironmentKey]
	if env == nil {
		return nil
	}

	content, err := renderRequirements(env)
	if err != nil {
		return err
	}

	// command_path is an absolute workspace path (translated during initialize); the
	// entry script reads requirements.yaml from its directory.
	dir := path.Dir(commandPath)
	client := m.client
	if client == nil {
		client, err = filer.NewWorkspaceFilesClient(b.WorkspaceClient(ctx), dir)
		if err != nil {
			return err
		}
	}

	log.Debugf(ctx, "writing %s for job %s task %s to %s", requirementsFileName, jobName, task.TaskKey, dir)
	if err := client.Write(ctx, requirementsFileName, bytes.NewReader(content), filer.OverwriteIfExists, filer.CreateParentDirectories); err != nil {
		return fmt.Errorf("failed to upload %s next to command_path %q: %w", requirementsFileName, commandPath, err)
	}
	return nil
}

// renderRequirements builds the requirements.yaml content from a serverless
// environment spec: version from environment_version (or the legacy client field),
// dependencies from its pip dependency list.
func renderRequirements(env *compute.Environment) ([]byte, error) {
	version := env.EnvironmentVersion
	if version == "" {
		version = env.Client
	}
	spec := requirementsSpec{
		Version:      version,
		Dependencies: env.Dependencies,
	}
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(spec); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// environmentsByKey indexes a job's environments by their key for task lookup.
func environmentsByKey(envs []jobs.JobEnvironment) map[string]*compute.Environment {
	out := make(map[string]*compute.Environment, len(envs))
	for i := range envs {
		if envs[i].Spec != nil {
			out[envs[i].EnvironmentKey] = envs[i].Spec
		}
	}
	return out
}
