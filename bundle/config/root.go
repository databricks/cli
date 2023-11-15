package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/config"
	"github.com/databricks/cli/libs/config/convert"
	"github.com/databricks/cli/libs/config/merge"
	"github.com/databricks/cli/libs/config/yamlloader"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Root struct {
	value config.Value
	diags diag.Diagnostics

	depth int

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
	Workspace Workspace `json:"workspace,omitempty"`

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
	Sync Sync `json:"sync,omitempty"`

	// RunAs section allows to define an execution identity for jobs and pipelines runs
	RunAs *jobs.JobRunAs `json:"run_as,omitempty"`

	Experimental *Experimental `json:"experimental,omitempty"`

	// Permissions section allows to define permissions which will be
	// applied to all resources defined in bundle
	Permissions []resources.Permission `json:"permissions,omitempty"`
}

// Load loads the bundle configuration file at the specified path.
func Load(path string) (*Root, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var r Root
	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", path, err)
	}

	// Normalize dynamic configuration tree according to configuration type.
	v, diags := convert.Normalize(r, v)

	// Convert normalized configuration tree to typed configuration.
	err = convert.ToTyped(&r, v)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", path, err)
	}

	r.diags = diags

	// Store dynamic configuration for later reference (e.g. location information on all nodes).
	r.value = v

	// if r.Environments != nil && r.Targets != nil {
	// 	return nil, fmt.Errorf("both 'environments' and 'targets' are specified, only 'targets' should be used: %s", path)
	// }

	// if r.Environments != nil {
	// 	//TODO: add a command line notice that this is a deprecated option.
	// 	r.Targets = r.Environments
	// }

	r.Path = filepath.Dir(path)
	// r.SetConfigFilePath(path)

	// _, err = r.Resources.VerifyUniqueResourceIdentifiers()
	return &r, err
}

func (r *Root) MarkMutatorEntry() {
	r.depth++

	// Many test cases initialize a config as a Go struct literal.
	// The zero-initialized value for [wasLoaded] will be false,
	// and indicates we need to populate [r.value].
	if !r.value.IsValid() {
		nv, err := convert.FromTyped(r, config.NilValue)
		if err != nil {
			panic(err)
		}

		r.value = nv
	}

	// If we are entering a mutator at depth 1, we need to convert
	// the dynamic configuration tree to typed configuration.
	if r.depth == 1 {
		// Always run ToTyped upon entering a mutator.
		// Convert normalized configuration tree to typed configuration.
		err := convert.ToTyped(r, r.value)
		if err != nil {
			panic(err)
		}
	} else {
		nv, err := convert.FromTyped(r, config.NilValue)
		if err != nil {
			panic(err)
		}

		r.value = nv
	}
}

func (r *Root) MarkMutatorExit() {
	r.depth--

	// If we are exiting a mutator at depth 0, we need to convert
	// the typed configuration to a dynamic configuration tree.
	if r.depth == 0 {
		nv, err := convert.FromTyped(r, config.NilValue)
		if err != nil {
			panic(err)
		}

		r.value = nv
	}
}

func (r *Root) Diagnostics() diag.Diagnostics {
	return r.diags
}

// SetConfigFilePath configures the path that its configuration
// was loaded from in configuration leafs that require it.
func (r *Root) SetConfigFilePath(path string) {
	panic("nope")

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
	panic("nope")

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

func (r *Root) Merge(other *Root) error {
	// // Merge diagnostics.
	// r.diags = append(r.diags, other.diags...)

	// err := r.Sync.Merge(r, other)
	// if err != nil {
	// 	return err
	// }
	// other.Sync = Sync{}

	// // TODO: when hooking into merge semantics, disallow setting path on the target instance.
	// other.Path = ""

	// Check for safe merge, protecting against duplicate resource identifiers
	err := r.Resources.VerifySafeMerge(&other.Resources)
	if err != nil {
		return err
	}

	// Merge dynamic configuration values.
	nv, err := merge.Merge(r.value, other.value)
	if err != nil {
		return err
	}

	r.value = nv

	// Convert normalized configuration tree to typed configuration.
	err = convert.ToTyped(r, r.value)
	if err != nil {
		panic(err)
	}

	// TODO: define and test semantics for merging.
	// return mergo.Merge(r, other, mergo.WithOverride)
	return nil
}

func (r *Root) MergeTargetOverrides(name string) error {
	var tmp config.Value
	var err error

	target := r.value.Get("targets").Get(name)
	if target == config.NilValue {
		return nil
	}

	mergeField := func(name string) error {
		tmp, err = merge.Merge(r.value.Get(name), target.Get(name))
		if err != nil {
			return err
		}

		r.value.MustMap()[name] = tmp
		return nil
	}

	if err = mergeField("bundle"); err != nil {
		return err
	}

	if err = mergeField("workspace"); err != nil {
		return err
	}

	if err = mergeField("artifacts"); err != nil {
		return err
	}

	if err = mergeField("resources"); err != nil {
		return err
	}

	if err = mergeField("sync"); err != nil {
		return err
	}

	// Convert normalized configuration tree to typed configuration.
	err = convert.ToTyped(r, r.value)
	if err != nil {
		panic(err)
	}

	if target.Permissions != nil {
		err = mergo.Merge(&r.Permissions, target.Permissions, mergo.WithAppendSlice)
		if err != nil {
			return err
		}
	}

	return nil
}
