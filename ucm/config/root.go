// Package config holds the typed and dynamic configuration tree for a ucm
// deployment. Mirrors the shape of bundle/config but targets Unity Catalog.
package config

import (
	"bytes"
	"context"
	"fmt"
	"os"

	"github.com/databricks/cli/libs/diag"
	"github.com/databricks/cli/libs/dyn"
	"github.com/databricks/cli/libs/dyn/convert"
	"github.com/databricks/cli/libs/dyn/merge"
	"github.com/databricks/cli/libs/dyn/yamlloader"
	"github.com/databricks/cli/libs/log"
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

	// Targets is set to nil by SelectTarget once a target has been merged.
	Targets map[string]*Target `json:"targets,omitempty"`
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

	return r.updateWithDynamicValue(root)
}
