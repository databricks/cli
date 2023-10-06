package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"github.com/ghodss/yaml"
	"github.com/imdario/mergo"
)

type ConfigFileNames []string

// FileNames contains allowed names of bundle configuration files.
var FileNames = ConfigFileNames{"databricks.yml", "databricks.yaml", "bundle.yml", "bundle.yaml"}

func (c ConfigFileNames) FindInPath(path string) (string, error) {
	result := ""
	var firstErr error

	for _, file := range c {
		filePath := filepath.Join(path, file)
		_, err := os.Stat(filePath)
		if err == nil {
			if result != "" {
				return "", fmt.Errorf("multiple bundle root configuration files found in %s", path)
			}
			result = filePath
		} else {
			if firstErr == nil {
				firstErr = err
			}
		}
	}

	if result == "" {
		return "", firstErr
	}
	return result, nil
}

type Root struct {
	// Path contains the directory path to the root of the bundle.
	// It is set when loading `databricks.yml`.
	Path string `json:"-" bundle:"readonly"`

	// Contains user defined variables
	Variables map[string]*variable.Variable `json:"variables,omitempty"`

	// Bundle contains details about this bundle, such as its name,
	// version of the spec (TODO), default cluster, default warehouse, etc.
	Bundle Bundle `json:"bundle,omitempty"`

	// Include specifies a list of patterns of file names to load and
	// merge into the this configuration. Only includes defined in the root
	// `databricks.yml` are processed. Defaults to an empty list.
	Include []string `json:"include,omitempty"`

	// Workspace contains details about the workspace to connect to
	// and paths in the workspace tree to use for this bundle.
	Workspace *Workspace `json:"workspace,omitempty"`

	// Artifacts contains a description of all code artifacts in this bundle.
	Artifacts Artifacts `json:"artifacts,omitempty"`

	// Resources contains a description of all Databricks resources
	// to deploy in this bundle (e.g. jobs, pipelines, etc.).
	Resources Resources `json:"resources,omitempty"`

	// Targets can be used to differentiate settings and resources between
	// bundle deployment targets (e.g. development, staging, production).
	// If not specified, the code below initializes this field with a
	// single default-initialized target called "default".
	Targets map[string]*Target `json:"targets,omitempty"`

	// DEPRECATED. Left for backward compatibility with Targets
	Environments map[string]*Target `json:"environments,omitempty"`

	// Sync section specifies options for files synchronization
	Sync *Sync `json:"sync,omitempty"`

	// RunAs section allows to define an execution identity for jobs and pipelines runs
	RunAs *jobs.JobRunAs `json:"run_as,omitempty"`

	Experimental *Experimental `json:"experimental,omitempty"`
}

func Load(path string) (*Root, error) {
	var r Root

	stat, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	// If we were given a directory, assume this is the bundle root.
	if stat.IsDir() {
		path, err = FileNames.FindInPath(path)
		if err != nil {
			return nil, err
		}
	}

	if err := r.Load(path); err != nil {
		return nil, err
	}

	return &r, nil
}

// SetConfigFilePath configures the path that its configuration
// was loaded from in configuration leafs that require it.
func (r *Root) SetConfigFilePath(path string) {
	r.Resources.SetConfigFilePath(path)
	if r.Artifacts != nil {
		r.Artifacts.SetConfigFilePath(path)
	}

	if r.Targets != nil {
		for _, env := range r.Targets {
			if env == nil {
				continue
			}
			if env.Resources != nil {
				env.Resources.SetConfigFilePath(path)
			}
			if env.Artifacts != nil {
				env.Artifacts.SetConfigFilePath(path)
			}
		}
	}
}

// Initializes variables using values passed from the command line flag
// Input has to be a string of the form `foo=bar`. In this case the variable with
// name `foo` is assigned the value `bar`
func (r *Root) InitializeVariables(vars []string) error {
	for _, variable := range vars {
		parsedVariable := strings.SplitN(variable, "=", 2)
		if len(parsedVariable) != 2 {
			return fmt.Errorf("unexpected flag value for variable assignment: %s", variable)
		}
		name := parsedVariable[0]
		val := parsedVariable[1]

		if _, ok := r.Variables[name]; !ok {
			return fmt.Errorf("variable %s has not been defined", name)
		}
		err := r.Variables[name].Set(val)
		if err != nil {
			return fmt.Errorf("failed to assign %s to %s: %s", val, name, err)
		}
	}
	return nil
}

func (r *Root) Load(path string) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(raw, r)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", path, err)
	}

	if r.Environments != nil && r.Targets != nil {
		return fmt.Errorf("both 'environments' and 'targets' are specified, only 'targets' should be used: %s", path)
	}

	if r.Environments != nil {
		//TODO: add a command line notice that this is a deprecated option.
		r.Targets = r.Environments
	}

	r.Path = filepath.Dir(path)
	r.SetConfigFilePath(path)

	_, err = r.Resources.VerifyUniqueResourceIdentifiers()
	return err
}

func (r *Root) Merge(other *Root) error {
	err := r.Sync.Merge(r, other)
	if err != nil {
		return err
	}
	other.Sync = &Sync{}

	// TODO: when hooking into merge semantics, disallow setting path on the target instance.
	other.Path = ""

	// Check for safe merge, protecting against duplicate resource identifiers
	err = r.Resources.VerifySafeMerge(&other.Resources)
	if err != nil {
		return err
	}

	// TODO: define and test semantics for merging.
	return mergo.Merge(r, other, mergo.WithOverride)
}

func (r *Root) MergeTargetOverrides(target *Target) error {
	var err error

	// Target may be nil if it's empty.
	if target == nil {
		return nil
	}

	if target.Bundle != nil {
		err = mergo.Merge(&r.Bundle, target.Bundle, mergo.WithOverride)
		if err != nil {
			return err
		}
	}

	if target.Workspace != nil {
		err = mergo.Merge(r.Workspace, target.Workspace, mergo.WithOverride)
		if err != nil {
			return err
		}
	}

	if target.Artifacts != nil {
		err = mergo.Merge(&r.Artifacts, target.Artifacts, mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			return err
		}
	}

	if target.Resources != nil {
		err = mergo.Merge(&r.Resources, target.Resources, mergo.WithOverride, mergo.WithAppendSlice)
		if err != nil {
			return err
		}

		err = r.Resources.Merge()
		if err != nil {
			return err
		}
	}

	if target.Variables != nil {
		for k, v := range target.Variables {
			variable, ok := r.Variables[k]
			if !ok {
				return fmt.Errorf("variable %s is not defined but is assigned a value", k)
			}
			// we only allow overrides of the default value for a variable
			defaultVal := v
			variable.Default = &defaultVal
		}
	}

	if target.RunAs != nil {
		r.RunAs = target.RunAs
	}

	if target.Mode != "" {
		r.Bundle.Mode = target.Mode
	}

	if target.ComputeID != "" {
		r.Bundle.ComputeID = target.ComputeID
	}

	git := &r.Bundle.Git
	if target.Git.Branch != "" {
		git.Branch = target.Git.Branch
		git.Inferred = false
	}
	if target.Git.Commit != "" {
		git.Commit = target.Git.Commit
	}
	if target.Git.OriginURL != "" {
		git.OriginURL = target.Git.OriginURL
	}

	return nil
}
