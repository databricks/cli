// Package config holds the typed and dynamic configuration tree for a ucm
// deployment. Mirrors the shape of bundle/config but targets Unity Catalog.
package config

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/dynloc"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
	"github.com/databricks/cli/ucm/config/variable"
)

// Root is the root of the ucm.yml configuration tree.
//
// The shape intentionally mirrors bundle.config.Root: a private dyn.Value
// holds the normalized dynamic tree, and the typed fields are re-derived from
// it across every mutator apply via MarkMutatorEntry/Exit. This keeps location
// information and interpolation (landing in M1) cheap.
type Root struct {
	value dyn.Value
	depth int

	Ucm Ucm `json:"ucm"`

	Workspace Workspace `json:"workspace,omitempty"`
	Account   Account   `json:"account,omitempty"`

	Resources Resources `json:"resources,omitempty"`

	// Variables declared under the top-level `variables:` block.
	Variables map[string]*variable.Variable `json:"variables,omitempty"`
	// Include lists glob patterns of additional files to merge into the root
	// configuration. Only honored in the root ucm.yml (included files cannot
	// themselves declare an Include). Expanded by ProcessRootIncludes.
	Include []string `json:"include,omitempty"`

	// Targets is set to nil by SelectTarget once a target has been merged.
	Targets map[string]*Target `json:"targets,omitempty"`

	// Scripts binds user-defined shell commands to phase hooks. Keys are
	// ScriptHook values (pre_init, post_init, pre_deploy, post_deploy,
	// pre_destroy, post_destroy). Mirrors bundle.config.Root.Scripts.
	Scripts map[string]Script `json:"scripts,omitempty"`

	// Locations is an output-only field that holds configuration location
	// information for every path in the configuration tree. Populated by the
	// PopulateLocations mutator when `--include-locations` is set.
	Locations *dynloc.Locations `json:"__locations,omitempty" ucm:"internal"`
}

// GetLocations returns the source locations of the configuration value at the
// given dotted path, or nil if the path is not set.
func (r Root) GetLocations(path string) []dyn.Location {
	v, err := dyn.Get(r.value, path)
	if err != nil {
		return nil
	}
	return v.Locations()
}

// Load reads a ucm.yml file from disk.
func Load(path string) (*Root, diag.Diagnostics) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return LoadFromBytes(path, raw)
}

// LoadFromBytes parses raw YAML bytes into a Root. `path` is only used for
// diagnostic source locations.
func LoadFromBytes(path string, raw []byte) (*Root, diag.Diagnostics) {
	r := Root{}

	v, err := yamlloader.LoadYAML(path, bytes.NewBuffer(raw))
	if err != nil {
		return nil, diag.Errorf("failed to load %s: %v", path, err)
	}

	v, diags := convert.Normalize(r, v)

	if err := r.updateWithDynamicValue(v); err != nil {
		diags = diags.Extend(diag.Errorf("failed to load %s: %v", path, err))
		return nil, diags
	}
	return &r, diags
}

// Value returns the current dynamic configuration tree.
func (r *Root) Value() dyn.Value {
	return r.value
}

func (r *Root) initializeDynamicValue() error {
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
	depth := r.depth
	defer func() { r.depth = depth }()

	if err := convert.ToTyped(r, nv); err != nil {
		return err
	}
	r.value = nv
	return nil
}

// Mutate applies fn to the dynamic configuration tree and re-syncs typed
// fields.
func (r *Root) Mutate(fn func(dyn.Value) (dyn.Value, error)) error {
	if err := r.initializeDynamicValue(); err != nil {
		return err
	}
	nv, err := fn(r.value)
	if err != nil {
		return err
	}
	return r.updateWithDynamicValue(nv)
}

// Merge folds another Root's dynamic tree into this one (used to load included
// files in M1).
func (r *Root) Merge(other *Root) error {
	return r.Mutate(func(root dyn.Value) (dyn.Value, error) {
		return merge.Merge(root, other.value)
	})
}

// MarkMutatorEntry is invoked by ApplyContext before each mutator runs.
// It keeps the dynamic and typed trees consistent across mutator boundaries.
func (r *Root) MarkMutatorEntry(ctx context.Context) error {
	if err := r.initializeDynamicValue(); err != nil {
		return err
	}
	r.depth++

	if r.depth == 1 {
		if err := r.updateWithDynamicValue(r.value); err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}
	} else {
		nv, err := convert.FromTyped(r, r.value)
		if err != nil {
			log.Warnf(ctx, "unable to convert typed configuration to dynamic configuration: %v", err)
			return err
		}
		if err := r.updateWithDynamicValue(nv); err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}
	}
	return nil
}

// MarkMutatorExit is invoked by ApplyContext after each mutator runs.
func (r *Root) MarkMutatorExit(ctx context.Context) error {
	r.depth--
	if r.depth == 0 {
		nv, err := convert.FromTyped(r, r.value)
		if err != nil {
			log.Warnf(ctx, "unable to convert typed configuration to dynamic configuration: %v", err)
			return err
		}
		if err := r.updateWithDynamicValue(nv); err != nil {
			log.Warnf(ctx, "unable to convert dynamic configuration to typed configuration: %v", err)
			return err
		}
	}
	return nil
}

func mergeField(rv, ov dyn.Value, name string) (dyn.Value, error) {
	path := dyn.NewPath(dyn.Key(name))
	reference, _ := dyn.GetByPath(rv, path)
	override, _ := dyn.GetByPath(ov, path)

	var out dyn.Value
	var err error
	switch {
	case reference.IsValid() && override.IsValid():
		out, err = merge.Merge(reference, override)
		if err != nil {
			return dyn.InvalidValue, err
		}
	case reference.IsValid():
		out = reference
	case override.IsValid():
		out = override
	default:
		return rv, nil
	}
	return dyn.SetByPath(rv, path, out)
}

// MergeTargetOverrides merges the named target's overrides into the root
// configuration. Fields that should be overwritten (not deep-merged) are
// handled explicitly.
func (r *Root) MergeTargetOverrides(name string) error {
	root := r.value
	target, err := dyn.GetByPath(root, dyn.NewPath(dyn.Key("targets"), dyn.Key(name)))
	if err != nil {
		return err
	}

	for _, f := range []string{"workspace", "account", "resources"} {
		if root, err = mergeField(root, target, f); err != nil {
			return fmt.Errorf("failed to merge target=%s field=%s: %w", name, f, err)
		}
	}

	// Merge `variables`. Per-variable: target override replaces default or
	// lookup on the root definition (never deep-merged) so SetVariables sees
	// a consistent view.
	if v := target.Get("variables"); v.Kind() != dyn.KindInvalid {
		_, err = dyn.Map(v, ".", dyn.Foreach(func(p dyn.Path, tv dyn.Value) (dyn.Value, error) {
			varPath := dyn.MustPathFromString("variables").Append(p...)

			if d := tv.Get("default"); d.Kind() != dyn.KindInvalid {
				if root, err = dyn.SetByPath(root, varPath.Append(dyn.Key("default")), d); err != nil {
					return root, err
				}
				// Clear any root-level lookup when target pins a default.
				if root, err = dyn.SetByPath(root, varPath.Append(dyn.Key("lookup")), dyn.NilValue); err != nil {
					return root, err
				}
			}

			if l := tv.Get("lookup"); l.Kind() != dyn.KindInvalid {
				if root, err = dyn.SetByPath(root, varPath.Append(dyn.Key("lookup")), l); err != nil {
					return root, err
				}
				if root, err = dyn.SetByPath(root, varPath.Append(dyn.Key("default")), dyn.NilValue); err != nil {
					return root, err
				}
			}
			return root, nil
		}))
		if err != nil {
			return fmt.Errorf("failed to merge target=%s variables: %w", name, err)
		}
	}

	return r.updateWithDynamicValue(root)
}

// InitializeVariables assigns variable values from CLI-flag pairs of the form
// "key=value". Mirrors bundle.config.Root.InitializeVariables: called from the
// CLI layer after the target merge so --var always wins over defaults.
func (r *Root) InitializeVariables(vars []string) error {
	for _, v := range vars {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("unexpected flag value for variable assignment: %s", v)
		}
		name, val := parts[0], parts[1]

		if _, ok := r.Variables[name]; !ok {
			return fmt.Errorf("variable %s has not been defined", name)
		}
		if r.Variables[name].IsComplex() {
			return fmt.Errorf("setting variables of complex type via --var flag is not supported: %s", name)
		}
		if err := r.Variables[name].Set(val); err != nil {
			return fmt.Errorf("failed to assign %s to %s: %s", val, name, err)
		}
	}
	return nil
}
