// Package trainyaml parses the AIR CLI's train.yaml launch config and converts
// it into a DABs job resource that uses an AI Runtime task. It exists so users
// who previously launched ephemeral runs with the AIR CLI can migrate onto a
// durable, DABs-managed job while keeping the familiar code_source experience.
package trainyaml

import (
	"bytes"
	"fmt"
	"os"

	"go.yaml.in/yaml/v3"
)

// Config is the subset of the AIR CLI train.yaml schema this migration
// understands. Fields that have no DABs equivalent are still parsed so we can
// warn about them rather than silently drop them (see Convert).
//
// Reference: universe/ai-compute/cli/cli/sdk/config.py (RunConfig and friends).
type Config struct {
	ExperimentName string  `yaml:"experiment_name"`
	Compute        Compute `yaml:"compute"`
	Command        string  `yaml:"command"`

	Environment *Environment `yaml:"environment"`
	CodeSource  *CodeSource  `yaml:"code_source"`

	EnvVariables map[string]string `yaml:"env_variables"`
	Secrets      map[string]string `yaml:"secrets"`

	MaxRetries                *int   `yaml:"max_retries"`
	TimeoutMinutes            *int   `yaml:"timeout_minutes"`
	MlflowRunName             string `yaml:"mlflow_run_name"`
	MlflowExperimentDirectory string `yaml:"mlflow_experiment_directory"`

	// Fields with no AI Runtime task equivalent. Retained so Convert can warn.
	Parameters       map[string]any `yaml:"parameters"`
	Priority         *int           `yaml:"priority"`
	UsagePolicyName  string         `yaml:"usage_policy_name"`
	UsagePolicyID    string         `yaml:"usage_policy_id"`
	IdempotencyToken string         `yaml:"idempotency_token"`
}

type Compute struct {
	NumAccelerators int    `yaml:"num_accelerators"`
	AcceleratorType string `yaml:"accelerator_type"`
	NodePoolID      string `yaml:"node_pool_id"`
	PoolName        string `yaml:"pool_name"`
}

// Environment mirrors the train.yaml environment block. Dependencies is
// polymorphic in the AIR schema (a requirements.yaml path or an inline list);
// we only support the inline list on the DABs serverless environment and report
// the string form as unsupported.
type Environment struct {
	Dependencies dependencies `yaml:"dependencies"`
	Version      string       `yaml:"version"`
	DockerImage  *DockerImage `yaml:"docker_image"`
}

type DockerImage struct {
	URL string `yaml:"url"`
}

type CodeSource struct {
	Type     string    `yaml:"type"`
	Snapshot *Snapshot `yaml:"snapshot"`
}

type Snapshot struct {
	RootPath     string   `yaml:"root_path"`
	RemoteVolume string   `yaml:"remote_volume"`
	IncludePaths []string `yaml:"include_paths"`
}

// dependencies captures the polymorphic environment.dependencies field. Exactly
// one of List / Path is set after unmarshaling.
type dependencies struct {
	List []string
	Path string
}

func (d *dependencies) UnmarshalYAML(node *yaml.Node) error {
	switch node.Kind {
	case yaml.ScalarNode:
		return node.Decode(&d.Path)
	case yaml.SequenceNode:
		return node.Decode(&d.List)
	default:
		return fmt.Errorf("environment.dependencies must be a string or a list, got %v", node.Kind)
	}
}

// Parse reads and decodes a train.yaml file. Unknown fields are rejected so a
// typo or an unsupported field surfaces to the user instead of being ignored.
func Parse(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", path, err)
	}
	return &cfg, nil
}
