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
	"github.com/databricks/cli/libs/dyn/dynloc"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Script struct {
	// Content of the script to be executed.
	Content string `json:"content"`
}

type Root struct {
	value dyn.Value
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
	// Note that this field is set to 'nil' by the SelectTarget mutator;
	// use bundle.Bundle.Target to access the selected target configuration.
	Targets map[string]*Target `json:"targets,omitempty"`

	// DEPRECATED. Left for backward compatibility with Targets
	Environments map[string]*Target `json:"environments,omitempty"`

	// Sync section specifies options for files synchronization
	Sync Sync `json:"sync,omitempty"`

	// RunAs section allows to define an execution identity for jobs and pipelines runs
	RunAs *jobs.JobRunAs `json:"run_as,omitempty"`

	// Presets applies preset transformations throughout the bundle, e.g.
	// adding a name prefix to deployed resources.
	Presets Presets `json:"presets,omitempty"`

	Experimental *Experimental `json:"experimental,omitempty"`

	// Permissions section allows to define permissions which will be
	// applied to all resources defined in bundle
	Permissions []resources.Permission `json:"permissions,omitempty"`

	// Locations is an output-only field that holds configuration location
	// information for every path in the configuration tree.
	Locations *dynloc.Locations `json:"__locations,omitempty" bundle:"internal"`

	Scripts map[string]Script `json:"scripts,omitempty"`
}

// Load loads the bundle configuration file at the specified path.
func Load(path string) (*Root, diag.Diagnostics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return LoadFromBytes(path, raw)
}

func LoadFromBytes(path string, raw []byte) (*Root, diag.Diagnostics) {
	r := Root{}

	// Load configuration tree from YAML.
	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(raw))
	if err != nil {
		return nil, diag.Errorf("failed to load %s: %v", path, err)
	}

	// Rewrite configuration tree where necessary.
	v, err = rewriteShorthands(v)
	if err != nil {
		return nil, diag.Errorf("failed to rewrite %s: %v", path, err)
	}

	// Normalize dynamic configuration tree according to configuration type.
	v, diags := convert.Normalize(r, v)

	// Convert normalized configuration tree to typed configuration.
	err = r.updateWithDynamicValue(v)
	if err != nil {
		diags = diags.Extend(diag.Errorf("failed to load %s: %v", path, err))
		return nil, diags
	}
	return &r, diags
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
	depth := r.depth

	defer func() {
		r.depth = depth
	}()

	// Convert normalized configuration tree to typed configuration.
	err := convert.ToTyped(r, nv)
	if err != nil {
		return err
	}

	// Assign the normalized configuration tree.
	r.value = nv
	return nil
}

// Mutate applies a transformation to the dynamic configuration value of a Root object.
//
// Parameters:
// - fn: A function that mutates a dyn.Value object
//
// Example usage, setting bundle.deployment.lock.enabled to false:
//
//	err := b.Config.Mutate(func(v dyn.Value) (dyn.Value, error) {
//	    return dyn.Map(v, "bundle.deployment.lock", func(_ dyn.Path, v dyn.Value) (dyn.Value, error) {
//	        return dyn.Set(v, "enabled", dyn.V(false))
//	    })
//	})
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

		if r.Variables[name].IsComplex() {
			return fmt.Errorf("setting variables of complex type via --var flag is not supported: %s", name)
		}

		err := r.Variables[name].Set(val)
		if err != nil {
			return fmt.Errorf("failed to assign %s to %s: %s", val, name, err)
		}
	}
	return nil
}

func (r *Root) Merge(other *Root) error {
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
		"presets",
	} {
		if root, err = mergeField(root, target, f); err != nil {
			return fmt.Errorf("failed to merge target=%s field=%s: %w", name, f, err)
		}
	}

	// Merge `variables`. This field must be overwritten if set, not merged.
	if v := target.Get("variables"); v.Kind() != dyn.KindInvalid {
		_, err = dyn.Map(v, ".", dyn.Foreach(func(p dyn.Path, variable dyn.Value) (dyn.Value, error) {
			varPath := dyn.MustPathFromString("variables").Append(p...)

			vDefault := variable.Get("default")
			if vDefault.Kind() != dyn.KindInvalid {
				defaultPath := varPath.Append(dyn.Key("default"))
				root, err = dyn.SetByPath(root, defaultPath, vDefault)
			}

			vLookup := variable.Get("lookup")
			if vLookup.Kind() != dyn.KindInvalid {
				lookupPath := varPath.Append(dyn.Key("lookup"))
				root, err = dyn.SetByPath(root, lookupPath, vLookup)
			}

			return root, err
		}))
		if err != nil {
			return err
		}
	}

	// Merge `run_as`. This field must be overwritten if set, not merged.
	if v := target.Get("run_as"); v.Kind() != dyn.KindInvalid {
		root, err = dyn.Set(root, "run_as", v)
		if err != nil {
			return err
		}
	}

	// Below, we're setting fields on the bundle key, so make sure it exists.
	if root.Get("bundle").Kind() == dyn.KindInvalid {
		root, err = dyn.Set(root, "bundle", dyn.V(map[string]dyn.Value{}))
		if err != nil {
			return err
		}
	}

	// Merge `mode`. This field must be overwritten if set, not merged.
	if v := target.Get("mode"); v.Kind() != dyn.KindInvalid {
		root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("mode")), v)
		if err != nil {
			return err
		}
	}

	// Merge `cluster_id`. This field must be overwritten if set, not merged.
	if v := target.Get("cluster_id"); v.Kind() != dyn.KindInvalid {
		root, err = dyn.SetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("cluster_id")), v)
		if err != nil {
			return err
		}
	}

	// Merge `git`.
	if v := target.Get("git"); v.Kind() != dyn.KindInvalid {
		ref, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("bundle"), dyn.Key("git")))
		if err != nil {
			ref = dyn.V(map[string]dyn.Value{})
		}

		// Merge the override into the reference.
		out, err := merge.Merge(ref, v)
		if err != nil {
			return err
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

var allowedVariableDefinitions = []([]string){
	{"default", "type", "description"},
	{"default", "type"},
	{"default", "description"},
	{"lookup", "description"},
	{"default"},
	{"lookup"},
}

// isFullVariableOverrideDef checks if the given value is a full syntax varaible override.
// A full syntax variable override is a map with either 1 of 2 keys.
// If it's 2 keys, the keys should be "default" and "type".
// If it's 1 key, the key should be one of the following keys: "default", "lookup".
func isFullVariableOverrideDef(v dyn.Value) bool {
	mv, ok := v.AsMap()
	if !ok {
		return false
	}

	// If the map has more than 3 keys, it is not a full variable override.
	if mv.Len() > 3 {
		return false
	}

	for _, keys := range allowedVariableDefinitions {
		if len(keys) != mv.Len() {
			continue
		}

		// Check if the keys are the same.
		match := true
		for _, key := range keys {
			if _, ok := mv.GetByString(key); !ok {
				match = false
				break
			}
		}

		if match {
			return true
		}
	}

	return false
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
		if target.Get("variables").Kind() == dyn.KindInvalid {
			return target, nil
		}

		// For each variable, normalize its contents if it is a single string.
		return dyn.Map(target, "variables", dyn.Foreach(func(p dyn.Path, variable dyn.Value) (dyn.Value, error) {
			switch variable.Kind() {

			case dyn.KindString, dyn.KindBool, dyn.KindFloat, dyn.KindInt:
				// Rewrite the variable to a map with a single key called "default".
				// This conforms to the variable type. Normalization back to the typed
				// configuration will convert this to a string if necessary.
				return dyn.NewValue(map[string]dyn.Value{
					"default": variable,
				}, variable.Locations()), nil

			case dyn.KindMap, dyn.KindSequence:
				// If it's a full variable definition, leave it as is.
				if isFullVariableOverrideDef(variable) {
					return variable, nil
				}

				// Check if the original definition of variable has a type field.
				// If it has a type field, it means the shorthand is a value of a complex type.
				// Type might not be found if the variable overriden in a separate file
				// and configuration is not merged yet.
				typeV, err := dyn.GetByPath(v, p.Append(dyn.Key("type")))
				if err == nil && typeV.MustString() == "complex" {
					return dyn.NewValue(map[string]dyn.Value{
						"type":    typeV,
						"default": variable,
					}, variable.Locations()), nil
				}

				// If it's a shorthand, rewrite it to a full variable definition.
				return dyn.NewValue(map[string]dyn.Value{
					"default": variable,
				}, variable.Locations()), nil

			default:
				return variable, nil
			}
		}))
	}))
}

// validateVariableOverrides checks that all variables specified
// in the target override are also defined in the root.
func validateVariableOverrides(root, target dyn.Value) (err error) {
	var rv map[string]variable.Variable
	var tv map[string]variable.Variable

	// Collect variables from the root.
	if v := root.Get("variables"); v.Kind() != dyn.KindInvalid {
		err = convert.ToTyped(&rv, v)
		if err != nil {
			return fmt.Errorf("unable to collect variables from root: %w", err)
		}
	}

	// Collect variables from the target.
	if v := target.Get("variables"); v.Kind() != dyn.KindInvalid {
		err = convert.ToTyped(&tv, v)
		if err != nil {
			return fmt.Errorf("unable to collect variables from target: %w", err)
		}
	}

	// Check that all variables in the target exist in the root.
	for k := range tv {
		if _, ok := rv[k]; !ok {
			return fmt.Errorf("variable %s is not defined but is assigned a value", k)
		}
	}

	return nil
}

// Best effort to get the location of configuration value at the specified path.
// This function is useful to annotate error messages with the location, because
// we don't want to fail with a different error message if we cannot retrieve the location.
func (r Root) GetLocation(path string) dyn.Location {
	v, err := dyn.Get(r.value, path)
	if err != nil {
		return dyn.Location{}
	}
	return v.Location()
}

// Get all locations of the configuration value at the specified path. We need both
// this function and it's singular version (GetLocation) because some diagnostics just need
// the primary location and some need all locations associated with a configuration value.
func (r Root) GetLocations(path string) []dyn.Location {
	v, err := dyn.Get(r.value, path)
	if err != nil {
		return nil
	}
	return v.Locations()
}

// Value returns the dynamic configuration value of the root object. This value
// is the source of truth and is kept in sync with values in the typed configuration.
func (r Root) Value() dyn.Value {
	return r.value
}
