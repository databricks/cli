package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Root struct {
	value dyn.Value
	diags diag.Diagnostics
	depth int

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

	r := Root{}

	// Load configuration tree from YAML.
	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(raw))
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", path, err)
	}

	// Rewrite configuration tree where necessary.
	v, err = rewriteShorthands(v)
	if err != nil {
		return nil, fmt.Errorf("failed to rewrite %s: %w", path, err)
	}

	// Normalize dynamic configuration tree according to configuration type.
	v, diags := convert.Normalize(r, v)

	// Keep track of diagnostics (warnings and errors in the schema).
	// We delay acting on diagnostics until we have loaded all
	// configuration files and merged them together.
	r.diags = diags

	// Convert normalized configuration tree to typed configuration.
	err = r.updateWithDynamicValue(v)
	if err != nil {
		return nil, fmt.Errorf("failed to load %s: %w", path, err)
	}

	_, err = r.Resources.VerifyUniqueResourceIdentifiers()
	return &r, err
}

func (r *Root) initializeDynamicValue() error {
	// Many test cases initialize a config as a Go struct literal.
	// The value will be invalid and we need to populate it from the typed configuration.
	if r.value.IsValid() {
		return nil
	}

	nv, err := convert.FromTyped(r, dyn.NilValue)
	if err != nil {
		return err
	}

	r.value = nv
	return nil
}

func (r *Root) updateWithDynamicValue(nv dyn.Value) error {
	// Hack: restore state; it may be cleared by [ToTyped] if
	// the configuration equals nil (happens in tests).
	diags := r.diags
	depth := r.depth

	defer func() {
		r.diags = diags
		r.depth = depth
	}()

	// Convert normalized configuration tree to typed configuration.
	err := convert.ToTyped(r, nv)
	if err != nil {
		return err
	}

	// Assign the normalized configuration tree.
	r.value = nv

	// Assign config file paths after converting to typed configuration.
	r.ConfigureConfigFilePath()
	return nil
}

func (r *Root) Mutate(fn func(dyn.Value) (dyn.Value, error)) error {
	err := r.initializeDynamicValue()
	if err != nil {
		return err
	}
	nv, err := fn(r.value)
	if err != nil {
		return err
	}
	err = r.updateWithDynamicValue(nv)
	if err != nil {
		return err
	}
	return nil
}

func (r *Root) MarkMutatorEntry(ctx context.Context) error {
	err := r.initializeDynamicValue()
	if err != nil {
		return err
	}

	r.depth++

	// If we are entering a mutator at depth 1, we need to convert
	// the dynamic configuration tree to typed configuration.
	if r.depth == 1 {
		// Always run ToTyped upon entering a mutator.
		// Convert normalized configuration tree to typed configuration.
		err := r.updateWithDynamicValue(r.value)
		if err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}

	} else {
		nv, err := convert.FromTyped(r, r.value)
		if err != nil {
			log.Warnf(ctx, "unable to convert typed configuration to dynamic configuration: %v", err)
			return err
		}

		// Re-run ToTyped to ensure that no state is piggybacked
		err = r.updateWithDynamicValue(nv)
		if err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}
	}

	return nil
}

func (r *Root) MarkMutatorExit(ctx context.Context) error {
	r.depth--

	// If we are exiting a mutator at depth 0, we need to convert
	// the typed configuration to a dynamic configuration tree.
	if r.depth == 0 {
		nv, err := convert.FromTyped(r, r.value)
		if err != nil {
			log.Warnf(ctx, "unable to convert typed configuration to dynamic configuration: %v", err)
			return err
		}

		// Re-run ToTyped to ensure that no state is piggybacked
		err = r.updateWithDynamicValue(nv)
		if err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}
	}

	return nil
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
	// Merge diagnostics.
	r.diags = append(r.diags, other.diags...)

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
	root := r.value
	target, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("targets"), dyn.Key(name)))
	if err != nil {
		return err
	}

	// Confirm validity of variable overrides.
	err = validateVariableOverrides(root, target)
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
		"variables",
	} {
		if root, err = mergeField(root, target, f); err != nil {
			return err
		}
	}

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

	// Convert normalized configuration tree to typed configuration.
	return r.updateWithDynamicValue(root)
}

// rewriteShorthands performs lightweight rewriting of the configuration
// tree where we allow users to write a shorthand and must rewrite to the full form.
func rewriteShorthands(v dyn.Value) (dyn.Value, error) {
	if v.Kind() != dyn.KindMap {
		return v, nil
	}

	// For each target, rewrite the variables block.
	return dyn.Map(v, "targets", dyn.Foreach(func(_ dyn.Path, target dyn.Value) (dyn.Value, error) {
		// Confirm it has a variables block.
		if target.Get("variables") == dyn.NilValue {
			return target, nil
		}

		// For each variable, normalize its contents if it is a single string.
		return dyn.Map(target, "variables", dyn.Foreach(func(_ dyn.Path, variable dyn.Value) (dyn.Value, error) {
			if variable.Kind() != dyn.KindString {
				return variable, nil
			}

			// Rewrite the variable to a map with a single key called "default".
			// This conforms to the variable type.
			return dyn.NewValue(map[string]dyn.Value{
				"default": variable,
			}, variable.Location()), nil
		}))
	}))
}

// validateVariableOverrides checks that all variables specified
// in the target override are also defined in the root.
func validateVariableOverrides(root, target dyn.Value) (err error) {
	var rv map[string]variable.Variable
	var tv map[string]variable.Variable

	// Collect variables from the root.
	err = convert.ToTyped(&rv, root.Get("variables"))
	if err != nil {
		return fmt.Errorf("unable to collect variables from root: %w", err)
	}

	// Collect variables from the target.
	err = convert.ToTyped(&tv, target.Get("variables"))
	if err != nil {
		return fmt.Errorf("unable to collect variables from target: %w", err)
	}

	// Check that all variables in the target exist in the root.
	for k := range tv {
		if _, ok := rv[k]; !ok {
			return fmt.Errorf("variable %s is not defined but is assigned a value", k)
		}
	}

	return nil
}
