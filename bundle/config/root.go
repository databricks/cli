package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"maps"
	"os"
	"reflect"
	"slices"
	"strings"

	"github.com/databricks/cli/bundle/config/loctable"
	"github.com/databricks/cli/bundle/config/resources"
	"github.com/databricks/cli/bundle/config/variable"
	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/structs/structpath"
	"github.com/databricks/cli/libs/structs/structvar"
	"github.com/databricks/databricks-sdk-go/service/jobs"
	"go.yaml.in/yaml/v3"
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

// Load loads the bundle configuration file at the specified path.
func Load(path string) (*Root, diag.Diagnostics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.FromErr(err)
	}

	return LoadFromBytes(path, raw)
}

func LoadFromBytes(path string, raw []byte) (*Root, diag.Diagnostics) {
	node, err := structvar.ParseYAML(bytes.NewReader(raw))
	if err != nil {
		return nil, diag.Errorf("failed to load %s: %v", path, err)
	}

	// Rewrite configuration tree where necessary.
	rewriteShorthands(node)

	var r Root
	sv, diags, err := structvar.DecodeYAMLNode(path, node, &r, nil)
	if err != nil {
		if le, ok := errors.AsType[*structvar.LocationError](err); ok {
			return nil, diag.Diagnostics{{
				Severity:  diag.Error,
				Summary:   le.Summary,
				Locations: []diag.Location{le.Loc},
			}}
		}
		return nil, diags.Extend(diag.Errorf("failed to load %s: %v", path, err))
	}
	r.store(sv)
	return &r, diags
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
			return fmt.Errorf("failed to assign %s to %s: %s", val, name, err)
		}
	}
	return nil
}

// Merge merges the other configurations into this one, in order.
func (r *Root) Merge(others ...*Root) error {
	for _, other := range others {
		if err := r.MergeAt(nil, other.View()); err != nil {
			return err
		}
	}
	return nil
}

var bundleGitPath = structpath.MustParsePath("bundle.git")

func (r *Root) MergeTargetOverrides(name string) error {
	targetPath := structpath.NewPath(nil, "targets", name)
	target := r.View().Lookup(targetPath)
	if !target.IsValid() {
		return fmt.Errorf("target %s not found", name)
	}

	// Confirm validity of variable overrides.
	err := validateVariableOverrides(r.Variables, r.Targets[name])
	if err != nil {
		return err
	}

	// Merge fields that can be merged 1:1. Check all of them first so that a failed
	// merge leaves the configuration unchanged.
	fields := []string{
		"bundle",
		"workspace",
		"artifacts",
		"resources",
		"sync",
		"permissions",
		"presets",
	}
	for _, f := range fields {
		if err := structvar.CheckMerge(r.View().Get(f), target.Get(f)); err != nil {
			return fmt.Errorf("failed to merge target=%s field=%s: %w", name, f, err)
		}
	}
	if err := structvar.CheckMerge(r.View().Lookup(bundleGitPath), target.Get("git")); err != nil {
		return err
	}
	for _, f := range fields {
		if err := r.MergeAt(structpath.NewStringKey(nil, f), target.Get(f)); err != nil {
			return fmt.Errorf("failed to merge target=%s field=%s: %w", name, f, err)
		}
	}

	// Merge `variables`. This field must be overwritten if set, not merged.
	for varName, variable := range target.Get("variables").MapItems() {
		varPath := structpath.NewPath(nil, "variables", varName)

		if vDefault := variable.Get("default"); vDefault.IsValid() {
			if err := r.Assign(structpath.NewStringKey(varPath, "default"), vDefault); err != nil {
				return err
			}

			// If the target explicitly sets a default value, drop any lookup from the
			// root variable definition so SetVariables can assign this default.
			if err := r.Delete(structpath.NewStringKey(varPath, "lookup")); err != nil {
				return err
			}
		}

		if vLookup := variable.Get("lookup"); vLookup.IsValid() {
			if err := r.Assign(structpath.NewStringKey(varPath, "lookup"), vLookup); err != nil {
				return err
			}

			// If the target explicitly sets a lookup, drop any default value from the
			// root variable definition so lookup resolution remains authoritative.
			if err := r.Delete(structpath.NewStringKey(varPath, "default")); err != nil {
				return err
			}
		}
	}

	// Merge `run_as`. This field must be overwritten if set, not merged.
	if v := target.Get("run_as"); v.IsValid() {
		if err := r.Assign(structpath.NewStringKey(nil, "run_as"), v); err != nil {
			return err
		}
	}

	// Merge `mode`. This field must be overwritten if set, not merged.
	if v := target.Get("mode"); v.IsValid() {
		if err := r.Assign(structpath.MustParsePath("bundle.mode"), v); err != nil {
			return err
		}
	}

	// Merge `cluster_id`. This field must be overwritten if set, not merged.
	if v := target.Get("cluster_id"); v.IsValid() {
		if err := r.Assign(structpath.MustParsePath("bundle.cluster_id"), v); err != nil {
			return err
		}
	}

	// Merge `git`.
	return r.MergeAt(bundleGitPath, target.Get("git"))
}

var allowedVariableDefinitions = []([]string){
	{"default", "type", "description"},
	{"default", "type"},
	{"default", "description"},
	{"lookup", "description"},
	{"default"},
	{"lookup"},
}

// isFullVariableOverrideDef checks if a mapping with the given keys is a full syntax
// variable override. A full syntax variable override is a map with either 1 of 2 keys.
// If it's 2 keys, the keys should be "default" and "type".
// If it's 1 key, the key should be one of the following keys: "default", "lookup".
func isFullVariableOverrideDef(keys []string) bool {
	// If the map has more than 3 keys, it is not a full variable override.
	if len(keys) > 3 {
		return false
	}

	for _, allowed := range allowedVariableDefinitions {
		if len(allowed) != len(keys) {
			continue
		}

		// Check if the keys are the same.
		match := true
		for _, key := range allowed {
			if !slices.Contains(keys, key) {
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

// dealias returns the node an alias refers to, or node itself.
func dealias(node *yaml.Node) *yaml.Node {
	for node != nil && node.Kind == yaml.AliasNode {
		node = node.Alias
	}
	return node
}

// mappingValue returns the value of key in the YAML mapping node (following aliases).
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	node = dealias(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return dealias(node.Content[i+1])
		}
	}
	return nil
}

// expandMergeKeys returns a copy of the mapping node with "<<" merge keys replaced
// by the pairs they merge in (keys set explicitly take precedence), so the pairs
// can be rewritten individually.
func expandMergeKeys(node *yaml.Node) *yaml.Node {
	var merged, explicit []*yaml.Node
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value != "<<" || node.Content[i].ShortTag() != "!!merge" {
			explicit = append(explicit, node.Content[i], node.Content[i+1])
			continue
		}
		sources := []*yaml.Node{dealias(node.Content[i+1])}
		if sources[0].Kind == yaml.SequenceNode {
			sources = sources[0].Content
		}
		for _, src := range sources {
			if src = dealias(src); src.Kind == yaml.MappingNode {
				merged = append(merged, expandMergeKeys(src).Content...)
			}
		}
	}
	if merged == nil {
		return node
	}
	out := *node
	out.Content = explicit
	for i := 0; i+1 < len(merged); i += 2 {
		if mappingValue(&out, merged[i].Value) == nil {
			out.Content = append(out.Content, merged[i], merged[i+1])
		}
	}
	return &out
}

func mappingKeys(node *yaml.Node) []string {
	var keys []string
	for i := 0; i+1 < len(node.Content); i += 2 {
		keys = append(keys, node.Content[i].Value)
	}
	return keys
}

// mappingNode returns a mapping node with the given keys and values, at the location of at.
func mappingNode(at *yaml.Node, kvs ...any) *yaml.Node {
	out := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Line: at.Line, Column: at.Column}
	for i := 0; i < len(kvs); i += 2 {
		out.Content = append(out.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: kvs[i].(string)}, kvs[i+1].(*yaml.Node))
	}
	return out
}

// rewriteShorthands performs lightweight rewriting of the configuration
// tree where we allow users to write a shorthand and must rewrite to the full form.
func rewriteShorthands(root *yaml.Node) {
	targets := mappingValue(root, "targets")
	if targets == nil || targets.Kind != yaml.MappingNode {
		return
	}

	// For each target, rewrite the variables block.
	for i := 1; i < len(targets.Content); i += 2 {
		variables := mappingValue(targets.Content[i], "variables")
		if variables == nil || variables.Kind != yaml.MappingNode {
			continue
		}
		*variables = *expandMergeKeys(variables)

		for j := 0; j+1 < len(variables.Content); j += 2 {
			name := variables.Content[j].Value
			variable := dealias(variables.Content[j+1])
			switch {
			case variable.Kind == yaml.ScalarNode && variable.ShortTag() != "!!null":
				// Rewrite the variable to a map with a single key called "default".
				// This conforms to the variable type. Normalization back to the typed
				// configuration will convert this to a string if necessary.
				variables.Content[j+1] = mappingNode(variable, "default", variable)

			case variable.Kind == yaml.MappingNode || variable.Kind == yaml.SequenceNode:
				// If it's a full variable definition, leave it as is.
				if variable.Kind == yaml.MappingNode && isFullVariableOverrideDef(mappingKeys(variable)) {
					continue
				}

				// Check if the original definition of variable has a type field.
				// If it has a type field, it means the shorthand is a value of a complex type.
				// Type might not be found if the variable overridden in a separate file
				// and configuration is not merged yet.
				typeV := mappingValue(mappingValue(mappingValue(root, "variables"), name), "type")
				if typeV != nil && typeV.Value == "complex" {
					variables.Content[j+1] = mappingNode(variable, "type", typeV, "default", variable)
					continue
				}

				// If it's a shorthand, rewrite it to a full variable definition.
				variables.Content[j+1] = mappingNode(variable, "default", variable)
			default:
			}
		}
	}
}

// validateVariableOverrides checks that all variables specified
// in the target override are also defined in the root.
func validateVariableOverrides(root map[string]*variable.Variable, target *Target) error {
	if target == nil {
		return nil
	}
	for k := range target.Variables {
		if _, ok := root[k]; !ok {
			return fmt.Errorf("variable %s is not defined but is assigned a value", k)
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

// UpdateSequence records that the elements of the sequence at path were rebuilt from
// the old ones (see [structvar.StructVar.UpdateSequence]).
func (r *Root) UpdateSequence(path *structpath.PathNode, sources [][]int) {
	sv := r.vars()
	sv.UpdateSequence(path, sources)
	r.store(sv)
}

// LocationsAt returns all locations of the configuration value at the specified path.
func (r Root) LocationsAt(path *structpath.PathNode) []diag.Location {
	return r.locations.At(path)
}

// Best effort to get the location of configuration value at the specified path.
// This function is useful to annotate error messages with the location, because
// we don't want to fail with a different error message if we cannot retrieve the location.
func (r Root) GetLocation(path string) diag.Location {
	locs := r.GetLocations(path)
	if len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

// Get all locations of the configuration value at the specified path. We need both
// this function and it's singular version (GetLocation) because some diagnostics just need
// the primary location and some need all locations associated with a configuration value.
// A value without locations (e.g. set by a mutator) gets those of its closest ancestor
// that has some; use [Root.DefinitionLocation] to find where a value is defined.
func (r Root) GetLocations(path string) []diag.Location {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return nil
	}
	return r.GetLocationsOf(p)
}

// GetLocationsOf is [Root.GetLocations] for a path node.
func (r Root) GetLocationsOf(path *structpath.PathNode) []diag.Location {
	return r.locations.Nearest(path)
}

// GetLocationOf is [Root.GetLocation] for a path node.
func (r Root) GetLocationOf(path *structpath.PathNode) diag.Location {
	locs := r.GetLocationsOf(path)
	if len(locs) == 0 {
		return diag.Location{}
	}
	return locs[0]
}

// DefinitionLocation returns the primary location the value at path is defined at,
// or an empty location if it has none (e.g. it was set by a mutator).
func (r Root) DefinitionLocation(path string) diag.Location {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return diag.Location{}
	}
	locs := r.locations.At(p)
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

// IsReference reports whether the value at path is a pure variable reference in a
// field that cannot hold a string (its typed value is the zero value).
func (r Root) IsReference(path string) bool {
	p, err := structpath.ParsePath(path)
	if err != nil {
		return false
	}
	_, ok := r.refs[p.String()]
	return ok
}

// References returns all pure variable references in fields that cannot hold a string,
// keyed by path. References in string fields are part of their value.
func (r Root) References() iter.Seq2[*structpath.PathNode, string] {
	return func(yield func(*structpath.PathNode, string) bool) {
		for _, k := range slices.Sorted(maps.Keys(r.refs)) {
			p, err := structpath.ParsePath(k)
			if err == nil && !yield(p, r.refs[k]) {
				return
			}
		}
	}
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
