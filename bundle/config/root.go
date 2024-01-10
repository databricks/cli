package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Root struct {
	value dyn.Value
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
	Environments map[string]*Target `json:"environments,omitempty" bundle:"deprecated"`

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

	r.Path = filepath.Dir(path)
	// r.SetConfigFilePath(path)

	r.ConfigureConfigFilePath()

	_, err = r.Resources.VerifyUniqueResourceIdentifiers()
	return &r, err
}

func (r *Root) initializeDynamicValue() {
	// Many test cases initialize a config as a Go struct literal.
	// The value will be invalid and we need to populate it from the typed configuration.
	if r.value.IsValid() {
		return
	}

	nv, err := convert.FromTyped(r, dyn.NilValue)
	if err != nil {
		panic(err)
	}

	r.value = nv
}

func (r *Root) toTyped(v dyn.Value) error {
	// Hack: restore state; it may be cleared by [ToTyped] if
	// the configuration equals nil (happens in tests).
	value := r.value
	diags := r.diags
	depth := r.depth
	path := r.Path

	defer func() {
		r.value = value
		r.diags = diags
		r.depth = depth
		r.Path = path
	}()

	// Convert normalized configuration tree to typed configuration.
	err := convert.ToTyped(r, v)
	if err != nil {
		return err
	}

	return nil
}

func (r *Root) Mutate(fn func(dyn.Value) (dyn.Value, error)) error {
	r.initializeDynamicValue()
	nv, err := fn(r.value)
	if err != nil {
		return err
	}
	err = r.toTyped(nv)
	if err != nil {
		return err
	}
	r.value = nv

	// Assign config file paths after mutating the configuration.
	r.ConfigureConfigFilePath()

	return nil
}

func (r *Root) MarkMutatorEntry() {
	r.initializeDynamicValue()
	r.depth++

	// If we are entering a mutator at depth 1, we need to convert
	// the dynamic configuration tree to typed configuration.
	if r.depth == 1 {
		// Always run ToTyped upon entering a mutator.
		// Convert normalized configuration tree to typed configuration.
		err := r.toTyped(r.value)
		if err != nil {
			panic(err)
		}

		r.ConfigureConfigFilePath()

	} else {
		nv, err := convert.FromTyped(r, r.value)
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
		nv, err := convert.FromTyped(r, r.value)
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
func (r *Root) ConfigureConfigFilePath() {
	r.Resources.ConfigureConfigFilePath()
	if r.Artifacts != nil {
		r.Artifacts.ConfigureConfigFilePath()
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

	// // TODO: when hooking into merge semantics, disallow setting path on the target instance.
	// other.Path = ""

	// Check for safe merge, protecting against duplicate resource identifiers
	err := r.Resources.VerifySafeMerge(&other.Resources)
	if err != nil {
		return err
	}

	// Merge dynamic configuration values.
	return r.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return merge.Merge(root, other.value)
	})
}

func mergeField(rv, ov dyn.Value, name string) (dyn.Value, error) {
	path := dyn.NewPath(dyn.Key(name))
	reference, _ := dyn.GetByPath(rv, path)
	override, _ := dyn.GetByPath(ov, path)

	// Merge the override into the reference.
	var out dyn.Value
	var err error
	if reference.IsValid() && override.IsValid() {
		out, err = merge.Merge(reference, override)
		if err != nil {
			return dyn.InvalidValue, err
		}
	} else if reference.IsValid() {
		out = reference
	} else if override.IsValid() {
		out = override
	} else {
		return rv, nil
	}

	return dyn.SetByPath(rv, path, out)
}

func (r *Root) MergeTargetOverrides(name string) error {
	// var tmp dyn.Value
	var root = r.value
	var err error

	target, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("targets"), dyn.Key(name)))
	if err != nil {
		return err
	}

	// Merge fields that can be merged 1:1.
	for _, f := range []string{
		"bundle",
		"workspace",
		"artifacts",
		"resources",
		"sync",
		"permissions",
	} {
		if root, err = mergeField(root, target, f); err != nil {
			return err
		}
	}

	// Merge variables.
	// TODO(@pietern):

	// Merge `run_as`. This field must be overwritten if set, not merged.
	if v := target.Get("run_as"); v != dyn.NilValue {
		root, err = dyn.Set(root, "run_as", v)
		if err != nil {
			return err
		}
	}

	// Below, we're setting fields on the bundle key, so make sure it exists.
	if root.Get("bundle") == dyn.NilValue {
		root, err = dyn.Set(root, "bundle", dyn.NewValue(map[string]dyn.Value{}, dyn.Location{}))
		if err != nil {
			return err
		}
	}

	// Merge `mode`. This field must be overwritten if set, not merged.
	if v := target.Get("mode"); v != dyn.NilValue {
		root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("mode")), v)
		if err != nil {
			return err
		}
	}

	// Merge `compute_id`. This field must be overwritten if set, not merged.
	if v := target.Get("compute_id"); v != dyn.NilValue {
		root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("compute_id")), v)
		if err != nil {
			return err
		}
	}

	// Merge `git`.
	if v := target.Get("git"); v != dyn.NilValue {
		ref, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("git")))
		if err != nil {
			ref = dyn.NewValue(map[string]dyn.Value{}, dyn.Location{})
		}

		// Merge the override into the reference.
		out, err := merge.Merge(ref, v)
		if err != nil {
			return err
		}

		// If the branch was overridden, we need to clear the inferred flag.
		if branch := v.Get("branch"); branch != dyn.NilValue {
			out, err = dyn.SetByPath(out, dyn.NewPath(dyn.Key("inferred")), dyn.NewValue(false, dyn.Location{}))
			if err != nil {
				return err
			}
		}

		// Set the merged value.
		root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("git")), out)
		if err != nil {
			return err
		}
	}

	r.value = root

	// Convert normalized configuration tree to typed configuration.
	err = r.toTyped(r.value)
	if err != nil {
		panic(err)
	}

	r.ConfigureConfigFilePath()
	return nil
}
