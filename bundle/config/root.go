package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/loctable"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/jobs"
)

type Script struct {
	// Content of the script to be executed.
	Content string `json:"content"`

	// Env is a map of environment variables exported when running the script.
	// Values may reference ${bundle.*}, ${workspace.*}, ${var.*} (or
	// ${variables.*}); other prefixes are rejected at validation time.
	// Use this to pass bundle configuration into a script's shell environment
	// without polluting the script content with DABs interpolation syntax.
	Env map[string]string `json:"env,omitempty"`
}

type Root struct { //nolint:recvcheck // value receivers for read-only accessors, pointer for mutators
	// value holds the configuration tree while the configuration is loading
	// (see [Root.Loading]); the fields below are zero until then.
	value dyn.Value

	// refs holds the pure variable references in fields that cannot hold a string
	// (e.g. `max_retries: ${var.n}`, where the typed field is zero), keyed by path.
	refs map[string]string

	// locations holds the source locations of the values.
	locations *structvar.Locations

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
	Locations *loctable.Locations `json:"__locations,omitempty" bundle:"internal"`

	Scripts map[string]Script `json:"scripts,omitempty"`

	// Python configures loading of Python code defined with 'databricks-bundles' package.
	Python Python `json:"python,omitempty"`

	// Not supported, must not be set. Placeholder to give better diagnostics in for OSS pipelines case.
	Definitions any `json:"definitions,omitempty" bundle:"internal"`
}

// Load loads the bundle configuration file at the specified path into a typed
// configuration (see [LoadDyn] to load it for merging).
func Load(path string) (*Root, diag.Diagnostics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return LoadFromBytes(path, raw)
}

// LoadFromBytes is [Load] for the contents of a file.
func LoadFromBytes(path string, raw []byte) (*Root, diag.Diagnostics) {
	r, diags := LoadDynFromBytes(path, raw)
	if diags.HasError() {
		return nil, diags
	}
	finishDiags, err := r.FinishLoading()
	diags = diags.Extend(finishDiags)
	if err != nil {
		return nil, diags.Extend(diag.Errorf("failed to load %s: %v", path, err))
	}
	return r, diags
}

// LoadDyn loads the bundle configuration file at the specified path into a
// configuration that is loading (see [Root.Loading]), to be merged with others.
func LoadDyn(path string) (*Root, diag.Diagnostics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return LoadDynFromBytes(path, raw)
}

// LoadDynFromBytes is [LoadDyn] for the contents of a file.
func LoadDynFromBytes(path string, raw []byte) (*Root, diag.Diagnostics) {
	// Load configuration tree from YAML.
	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(raw))
	if err != nil {
		if le, ok := errors.AsType[*yamlloader.LocationError](err); ok {
			return nil, diag.Diagnostics{{
				Severity:  diag.Error,
				Summary:   le.Summary,
				Locations: []diag.Location{le.Loc},
			}}
		}
		return nil, diag.Errorf("failed to load %s: %v", path, err)
	}

	// Rewrite configuration tree where necessary.
	v, err = rewriteShorthands(v)
	if err != nil {
		return nil, diag.Errorf("failed to rewrite %s: %v", path, err)
	}

	// Normalize dynamic configuration tree according to configuration type.
	v, diags := convert.Normalize(Root{}, v)
	if !v.IsValid() {
		// An empty document is null; keep the configuration loading.
		v = dyn.NilValue
	}
	return &Root{value: v}, diags
}

// LoadFromReader decodes the configuration in r, recording the locations mapLocations
// returns (if not nil) instead of the locations in the file.
func LoadFromReader(path string, r io.Reader, mapLocations structvar.LocationMapper) (*Root, diag.Diagnostics, error) {
	node, err := structvar.ParseYAML(r)
	if err != nil {
		return nil, nil, err
	}
	var root Root
	sv, diags, err := structvar.DecodeYAMLNode(path, node, &root, mapLocations)
	if err != nil {
		return nil, diags, err
	}
	root.store(sv)
	return &root, diags, nil
}

// Loading reports whether the configuration is loading: it is a configuration tree
// that the load phase merges (includes, target overrides) and only [Root.Value],
// [Root.Mutate], [Root.Merge] and [Root.MergeTargetOverrides] may be used. The typed
// fields are zero until [Root.FinishLoading] converts the tree into them.
func (r *Root) Loading() bool {
	return r.value.IsValid()
}

// Value returns the configuration tree of a loading configuration
// (invalid if it is not loading).
func (r *Root) Value() dyn.Value {
	return r.value
}

// startLoading makes an empty configuration loading. A bundle's configuration starts
// out empty and the entry point merges the loaded configuration into it.
func (r *Root) startLoading() error {
	if r.Loading() {
		return nil
	}
	if !reflect.ValueOf(r).Elem().IsZero() {
		return errors.New("cannot load into a typed configuration")
	}
	r.value = dyn.V(map[string]dyn.Value{})
	return nil
}

// Mutate applies fn to the configuration tree of a loading configuration.
func (r *Root) Mutate(fn func(dyn.Value) (dyn.Value, error)) error {
	if err := r.startLoading(); err != nil {
		return err
	}
	nv, err := fn(r.value)
	if err != nil {
		return err
	}
	r.value = nv
	return nil
}

// FinishLoading converts the configuration tree of a loading configuration into the
// typed configuration, its references and locations, once. The diagnostics explain
// values that could not be converted and were dropped.
func (r *Root) FinishLoading() (diag.Diagnostics, error) {
	if !r.Loading() {
		return nil, nil
	}
	var out Root
	sv, diags, err := structvar.DecodeDyn(r.value, &out)
	if err != nil {
		return diags, err
	}
	*r = out
	r.store(sv)
	return diags, nil
}

// vars returns the configuration as a [structvar.StructVar]; use [Root.store] to
// keep changes made to its references and locations.
func (r *Root) vars() *structvar.StructVar {
	return &structvar.StructVar{Value: r, Refs: r.refs, Locations: r.locations}
}

func (r *Root) store(sv *structvar.StructVar) {
	r.refs = sv.Refs
	r.locations = sv.Locations
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
			return fmt.Errorf("failed to assign %s to %s: %w", val, name, err)
		}
	}
	return nil
}

// Merge merges the other loading configurations into this one, in order.
func (r *Root) Merge(others ...*Root) error {
	if err := r.startLoading(); err != nil {
		return err
	}
	for _, other := range others {
		nv, err := merge.Merge(r.value, other.value)
		if err != nil {
			return err
		}
		r.value = nv
	}
	return nil
}

// HasTarget reports whether the loading configuration defines the target name.
func (r *Root) HasTarget(name string) bool {
	_, err := dyn.GetByPath(r.value, dyn.NewPath(dyn.Key("targets"), dyn.Key(name)))
	return err == nil
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
				if err != nil {
					return root, err
				}

				// If the target explicitly sets a default value, drop any lookup from the
				// root variable definition so SetVariables can assign this default.
				lookupPath := varPath.Append(dyn.Key("lookup"))
				root, err = dyn.SetByPath(root, lookupPath, dyn.NilValue)
			}

			vLookup := variable.Get("lookup")
			if vLookup.Kind() != dyn.KindInvalid {
				lookupPath := varPath.Append(dyn.Key("lookup"))
				root, err = dyn.SetByPath(root, lookupPath, vLookup)
				if err != nil {
					return root, err
				}

				// If the target explicitly sets a lookup, drop any default value from the
				// root variable definition so lookup resolution remains authoritative.
				defaultPath := varPath.Append(dyn.Key("default"))
				root, err = dyn.SetByPath(root, defaultPath, dyn.NilValue)
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

	r.value = root
	return nil
}

var allowedVariableDefinitions = []([]string){
	{"default", "type", "description"},
	{"default", "type"},
	{"default", "description"},
	{"lookup", "description"},
	{"default"},
	{"lookup"},
}

// isFullVariableOverrideDef checks if the given value is a full syntax variable override.
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
				// Type might not be found if the variable overridden in a separate file
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
func validateVariableOverrides(root, target dyn.Value) error {
	rv, _ := root.Get("variables").AsMap()
	tv, _ := target.Get("variables").AsMap()

	// Check that all variables in the target exist in the root.
	for _, p := range tv.Pairs() {
		if _, ok := rv.GetPair(p.Key); !ok {
			return fmt.Errorf("variable %s is not defined but is assigned a value", p.Key.MustString())
		}
	}

	return nil
}

// Set sets value at path (see [structvar.StructVar.Set]).
func (r *Root) Set(path *structpath.PathNode, value any) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.Set(path, value)
}

// SetReference records the pure reference ref at path, a field that cannot hold a string.
func (r *Root) SetReference(path *structpath.PathNode, ref string) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.SetReference(path, ref)
}

// Assign sets the value described by v at path, with its locations and references.
func (r *Root) Assign(path *structpath.PathNode, v structvar.View) error {
	_, err := r.Decode(path, v)
	return err
}

// Decode sets the value described by v at path, converting it to the type at path;
// the diagnostics explain values that could not be converted and were dropped.
func (r *Root) Decode(path *structpath.PathNode, v structvar.View) (diag.Diagnostics, error) {
	sv := r.vars()
	defer r.store(sv)
	return sv.Assign(path, v)
}

// Delete removes the value at path (see [structvar.StructVar.Delete]).
func (r *Root) Delete(path *structpath.PathNode) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.Delete(path)
}

// MergeAt merges the value described by v into the value at path (see [structvar.StructVar.Merge]).
func (r *Root) MergeAt(path *structpath.PathNode, v structvar.View) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.Merge(path, v)
}

// MergeElementsByKey merges the elements of the sequence at path that have the same
// key (see [structvar.StructVar.MergeElementsByKey]).
func (r *Root) MergeElementsByKey(path *structpath.PathNode, keyField string, keyFn func(structvar.View) string, sortKeys bool) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.MergeElementsByKey(path, keyField, keyFn, sortKeys)
}

// SetLocations sets the locations of the value at path and all values below it.
func (r *Root) SetLocations(path *structpath.PathNode, locs []diag.Location) {
	sv := r.vars()
	sv.SetLocations(path, locs)
	r.store(sv)
}

// CopyMetadata replaces the locations and references of the value at path with those in
// src (see [structvar.StructVar.CopyMetadata]).
func (r *Root) CopyMetadata(src *Root, path *structpath.PathNode) {
	sv := r.vars()
	sv.CopyMetadata(src.vars(), path)
	r.store(sv)
}

// MoveMetadata moves the locations and references of the value at path from to the
// path to (see [structvar.StructVar.MoveMetadata]).
func (r *Root) MoveMetadata(from, to *structpath.PathNode) {
	sv := r.vars()
	sv.MoveMetadata(from, to)
	r.store(sv)
}

// CopyMetadataAt copies the locations and references of the value at path from to the
// path to (see [structvar.StructVar.CopyMetadataAt]).
func (r *Root) CopyMetadataAt(from, to *structpath.PathNode) {
	sv := r.vars()
	sv.CopyMetadataAt(from, to)
	r.store(sv)
}

// UpdateSequence records that the elements of the sequence at path were rebuilt from
// the old ones (see [structvar.StructVar.UpdateSequence]).
func (r *Root) UpdateSequence(path *structpath.PathNode, sources [][]int) {
	sv := r.vars()
	sv.UpdateSequence(path, sources)
	r.store(sv)
}

// GetLocation is [Root.GetLocationOf] for a path string, like main's.
func (r Root) GetLocation(path string) diag.Location {
	return firstLocation(r.GetLocations(path))
}

// GetLocations is [Root.GetLocationsOf] for a path string, like main's.
func (r Root) GetLocations(path string) []diag.Location {
	if r.Loading() {
		v, err := dyn.Get(r.value, path)
		if err != nil {
			return nil
		}
		return v.Locations()
	}
	p, err := structpath.ParsePath(path)
	if err != nil {
		return nil
	}
	return r.GetLocationsOf(p)
}

// GetLocationsOf returns the locations of the configuration value at path or, if it
// has none (e.g. it was set by a mutator), those of its closest ancestor that has some.
// Use it to point diagnostics at the most specific place there is; use
// [Root.GetLocationsOfExact] to find where a value is written.
func (r Root) GetLocationsOf(path *structpath.PathNode) []diag.Location {
	if r.Loading() {
		return r.GetLocationsOfExact(path)
	}
	return r.locations.Nearest(path)
}

// GetLocationOf returns the primary location of [Root.GetLocationsOf].
func (r Root) GetLocationOf(path *structpath.PathNode) diag.Location {
	return firstLocation(r.GetLocationsOf(path))
}

// GetLocationsOfExact returns the locations the configuration value at path is written at,
// or none if it was set by a mutator.
func (r Root) GetLocationsOfExact(path *structpath.PathNode) []diag.Location {
	if r.Loading() {
		p, ok := dyn.FromStructPath(path)
		if !ok {
			return nil
		}
		v, err := dyn.GetByPath(r.value, p)
		if err != nil {
			return nil
		}
		return v.Locations()
	}
	return r.locations.Exact(path)
}

// GetLocationOfExact returns the primary location of [Root.GetLocationsOfExact].
func (r Root) GetLocationOfExact(path *structpath.PathNode) diag.Location {
	return firstLocation(r.GetLocationsOfExact(path))
}

func firstLocation(locs []diag.Location) diag.Location {
	if len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

// GetNodeAndType and returns parent resource node and type of the resource in direct backend.
// Examples:
//
//	"resources.jobs.foo.name" -> ("resources.jobs.foo", "jobs")
//	"resources.jobs.foo.permissions[0].level -> ("resources.jobs.foo.permissions", "jobs.permissions")
func GetNodeAndType(path *structpath.PathNode) (*structpath.PathNode, string) {
	if path.Len() < 3 {
		return nil, ""
	}

	if path.KeyAt(0) != "resources" {
		return nil, ""
	}

	if path.Len() >= 4 {
		if k := path.KeyAt(3); k == "permissions" || k == "grants" {
			return path.Prefix(4), path.KeyAt(1) + "." + k
		}
	}

	return path.Prefix(3), path.KeyAt(1)
}

// GetResourceTypeFromKey extracts the resource group from a resource path.
// For example, "resources.jobs.foo" returns "jobs".
// Returns empty string if the path is not in the expected format.
func GetResourceTypeFromKey(path string) string {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return ""
	}
	_, rType := GetNodeAndType(p)
	return rType
}

// GetResourceConfig returns the configuration object for a given resource path.
// The path should be in the format "resources.group.name" (e.g., "resources.jobs.foo").
// The returned value is a pointer to the concrete struct that represents that resource type.
// When the path is invalid or resource is not found, the second return value is false.
func (r *Root) GetResourceConfig(path string) (any, error) {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return nil, err
	}

	// Extract and validate the resource group from the path
	node, resourceType := GetNodeAndType(p)
	if resourceType == "" {
		return nil, fmt.Errorf("path does not correspond to resource: %q", path)
	}

	// Resolve the Go type that represents a single resource in this group.
	typ, ok := ResourcesTypes[resourceType]
	if !ok {
		return nil, fmt.Errorf("no such resource type in the config: %q", resourceType)
	}

	// Copy the value, so that the caller can't change the configuration through it.
	v := r.View().Lookup(node)
	if !v.IsValid() {
		return nil, nil
	}

	typedConfigPtr := reflect.New(typ)

	_, err = (&structvar.StructVar{Value: typedConfigPtr.Interface()}).Assign(nil, v)
	if err != nil {
		return nil, fmt.Errorf("cannot convert config to %s: %w", typ.String(), err)
	}

	return typedConfigPtr.Interface(), nil
}

// IsReferenceAt is [Root.IsReference] for a path node.
func (r Root) IsReferenceAt(path *structpath.PathNode) bool {
	_, ok := r.refs[path.String()]
	return ok
}

// ReferenceAt returns the pure variable reference held by the field at path that
// cannot hold a string, or "" if there is none.
func (r Root) ReferenceAt(path *structpath.PathNode) string {
	return r.refs[path.String()]
}

// References returns all pure variable references in fields that cannot hold a string,
// keyed by path. References in string fields are part of their value.
func (r Root) References() iter.Seq2[*structpath.PathNode, string] {
	return r.ReferencesUnder(nil)
}

// ReferencesUnder returns the pure variable references (see [Root.References]) at path
// and below it, in path order. References below a value that a mutator removed (e.g. a
// cleared new_cluster) are skipped: only those in the typed value are returned.
func (r Root) ReferencesUnder(path *structpath.PathNode) iter.Seq2[*structpath.PathNode, string] {
	return func(yield func(*structpath.PathNode, string) bool) {
		prefix := path.String()
		var keys []string
		for k := range r.refs {
			if prefix == "" || k == prefix || strings.HasPrefix(k, prefix+".") || strings.HasPrefix(k, prefix+"[") {
				keys = append(keys, k)
			}
		}
		slices.Sort(keys)
		view := r.View()
		for _, k := range keys {
			p, err := structpath.ParsePath(k)
			if err != nil {
				continue
			}
			// The view shows a present reference as a string. A reference below or at a
			// nil pointer (even a forced one), a missing map key or an index out of range
			// is gone.
			if view.Lookup(p).Kind() != structvar.KindString {
				continue
			}
			if !yield(p, r.refs[k]) {
				return
			}
		}
	}
}

// LocationTree returns the tree of the source locations of the configuration values.
func (r *Root) LocationTree() *structvar.Locations {
	return r.locations
}

// View returns the read-only view of the configuration tree.
func (r *Root) View() structvar.View {
	return structvar.NewView(r, r.refs, r.locations)
}

// Override replaces the configuration with the result of plan (see [structvar.PlanOverride]).
func (r *Root) Override(plan *structvar.OverridePlan) error {
	sv := r.vars()
	defer r.store(sv)
	return sv.Override(plan)
}

// DecodeLoading converts the value at the path of keys in the configuration tree of a
// loading configuration into dst, a pointer to a typed value; dst is left unchanged
// if there is no value there.
func (r *Root) DecodeLoading(dst any, keys ...string) error {
	v := r.value
	for _, k := range keys {
		v = v.Get(k)
	}
	if !v.IsValid() {
		return nil
	}
	_, _, err := structvar.DecodeDyn(v, dst)
	return err
}
